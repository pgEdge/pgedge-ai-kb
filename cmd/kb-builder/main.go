/*-------------------------------------------------------------------------
 *
 * pgEdge AI Knowledgebase Builder
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 *-------------------------------------------------------------------------
 */

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pgEdge/pgedge-ai-kb/internal/kbchunker"
	"github.com/pgEdge/pgedge-ai-kb/internal/kbconfig"
	"github.com/pgEdge/pgedge-ai-kb/internal/kbconverter"
	"github.com/pgEdge/pgedge-ai-kb/internal/kbdatabase"
	"github.com/pgEdge/pgedge-ai-kb/internal/kbembed"
	"github.com/pgEdge/pgedge-ai-kb/internal/kbsource"
	"github.com/pgEdge/pgedge-ai-kb/internal/kbtypes"
	"github.com/spf13/cobra"
)

var (
	configFile           string
	databasePath         string
	skipUpdates          bool
	addMissingEmbeddings bool
	clearEmbeddings      string
	maxRetries           int
)

var rootCmd = &cobra.Command{
	Use:   "pgedge-ai-kb-builder",
	Short: "pgEdge AI Knowledgebase Builder - Build searchable documentation databases",
	Long: `pgedge-ai-kb-builder processes documentation from various sources (Git
repositories or local paths) and builds a searchable SQLite database with vector
embeddings for use with retrieval-augmented AI tools.

The tool converts documents from multiple formats (Markdown, HTML, RST, SGML),
chunks them intelligently, generates embeddings using multiple providers (OpenAI,
Voyage, Gemini, Ollama), and stores everything in an optimized SQLite database.`,
	RunE: run,
}

func init() {
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "",
		"Path to configuration file (default: looks in binary directory)")
	rootCmd.Flags().StringVarP(&databasePath, "database", "d", "",
		"Path to output SQLite database (overrides config file)")
	rootCmd.Flags().BoolVar(&skipUpdates, "skip-updates", false,
		"Skip git pull updates for existing repositories")
	rootCmd.Flags().BoolVar(&addMissingEmbeddings, "add-missing-embeddings", false,
		"Add missing embeddings to existing database instead of rebuilding")
	rootCmd.Flags().StringVar(&clearEmbeddings, "clear-embeddings", "",
		"Clear embeddings for specified provider (openai, voyage, ollama, or gemini)")
	rootCmd.Flags().IntVar(&maxRetries, "max-retries", 5,
		"Maximum number of retries for transient embedding API errors (0 = unlimited)")
}

func main() {
	// Let cobra handle errors and exit codes
	// Usage is shown for flag parse errors, but suppressed for runtime errors (via cmd.SilenceUsage in run())
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Suppress usage for runtime errors (flags have already been parsed by this point)
	cmd.SilenceUsage = true

	// Load configuration
	if configFile == "" {
		// Use default config file in binary directory
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}
		configFile = filepath.Join(filepath.Dir(exePath), "pgedge-ai-kb-builder.yaml")
	}

	fmt.Printf("Loading configuration from: %s\n", configFile)
	config, err := kbconfig.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override database path if specified on command line
	if databasePath != "" {
		config.DatabasePath = databasePath
	}

	// If --add-missing-embeddings is specified, run that and exit
	if addMissingEmbeddings {
		return runAddMissingEmbeddings(config)
	}

	// If --clear-embeddings is specified, run that and exit
	if clearEmbeddings != "" {
		return runClearEmbeddings(config, clearEmbeddings)
	}

	targets := config.EnabledTargets()

	fmt.Printf("Doc source path: %s\n", config.DocSourcePath)
	fmt.Printf("Number of sources: %d\n", len(config.Sources))
	labels := make([]string, len(targets))
	for i, t := range targets {
		labels[i] = t.Label
	}
	fmt.Printf("Enabled embedding providers: %v\n", labels)
	fmt.Println("Output databases:")
	for _, t := range targets {
		fmt.Printf("  - %s (%s): %s\n", t.Label, t.Model, config.DatabasePathFor(t))
	}

	// Fetch all documentation sources once; every target reuses them.
	fmt.Println("\n=== Fetching Documentation Sources ===")
	if skipUpdates {
		fmt.Println("Note: Skipping git pull updates for existing repositories")
	}
	sources, err := kbsource.FetchAll(config, skipUpdates)
	if err != nil {
		return fmt.Errorf("failed to fetch sources: %w", err)
	}

	// Build one self-contained database per target.
	for _, t := range targets {
		if err := buildTarget(config, t, sources); err != nil {
			return fmt.Errorf("failed to build %s database: %w", t.Label, err)
		}
	}

	return nil
}

// buildTarget builds a single provider/model database: it re-processes
// the (already-fetched) sources into chunks, inserts them, and generates
// embeddings for this target's provider only. Re-chunking per target is
// deterministic and CPU-only; embedding API calls happen once per
// provider regardless, so the per-target loop does not increase API cost.
func buildTarget(config *kbconfig.Config, t kbconfig.Target, sources []kbsource.SourceInfo) error {
	dbPath := config.DatabasePathFor(t)
	fmt.Printf("\n========================================\n")
	fmt.Printf("Building %s database (%s)\n  -> %s\n", t.Label, t.Model, dbPath)
	fmt.Printf("========================================\n")

	db, err := kbdatabase.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Process all documents (with incremental processing). Chunks are
	// inserted without embeddings FIRST so each chunk gets a row ID; the
	// embedding generator then persists embeddings incrementally per
	// batch, so a ctrl-C mid-run leaves the database recoverable via
	// --add-missing-embeddings.
	fmt.Println("\n=== Processing Documents ===")
	allChunks, err := processAllDocuments(sources, db)
	if err != nil {
		return fmt.Errorf("failed to process documents: %w", err)
	}
	fmt.Printf("\nTotal chunks created/reused: %d\n", len(allChunks))

	fmt.Println("\n=== Storing chunks in Database ===")
	if err := db.InsertChunks(allChunks); err != nil {
		return fmt.Errorf("failed to insert chunks: %w", err)
	}

	fmt.Println("\n=== Generating Embeddings ===")
	embedGen, err := kbembed.NewEmbeddingGenerator(config.ForProvider(t.Provider), db, maxRetries)
	if err != nil {
		return fmt.Errorf("failed to initialize embedding generator: %w", err)
	}
	embeddingErrors := embedGen.GenerateEmbeddings(allChunks)
	if len(embeddingErrors) > 0 {
		fmt.Println("\n⚠️  Warning: Some embedding providers failed:")
		for provider, err := range embeddingErrors {
			fmt.Printf("  - %s: %v\n", provider, err)
		}
		fmt.Println("\nContinuing with partial embeddings. Use --add-missing-embeddings later to complete them.")
	}

	fmt.Println("\n=== Database Statistics ===")
	stats, err := db.GetStats()
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}
	fmt.Printf("Total chunks: %v\n", stats["total_chunks"])
	fmt.Println("Projects:")
	projects, ok := stats["projects"].([]map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected projects format in stats")
	}
	for _, project := range projects {
		fmt.Printf("  - %s %s: %d chunks\n",
			project["name"], project["version"], project["chunks"])
	}

	fmt.Printf("\n✓ Knowledgebase successfully built: %s\n", dbPath)
	return nil
}

func runAddMissingEmbeddings(config *kbconfig.Config) error {
	for _, t := range config.EnabledTargets() {
		if err := addMissingForTarget(config, t); err != nil {
			return fmt.Errorf("failed to add missing %s embeddings: %w", t.Label, err)
		}
	}
	return nil
}

// addMissingForTarget backfills the missing embeddings for a single
// provider/model database.
func addMissingForTarget(config *kbconfig.Config, t kbconfig.Target) error {
	dbPath := config.DatabasePathFor(t)
	fmt.Printf("\nAdding missing %s embeddings to: %s\n", t.Label, dbPath)

	db, err := kbdatabase.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	fmt.Println("Loading existing chunks from database...")
	chunks, err := db.GetAllChunks()
	if err != nil {
		return fmt.Errorf("failed to load chunks: %w", err)
	}
	fmt.Printf("Loaded %d chunks\n", len(chunks))

	var needing []*kbtypes.Chunk
	for _, chunk := range chunks {
		if missingEmbedding(chunk, t.Provider) {
			needing = append(needing, chunk)
		}
	}

	if len(needing) == 0 {
		fmt.Printf("✓ All chunks already have %s embeddings\n", t.Label)
		return nil
	}
	fmt.Printf("Found %d chunks with missing %s embeddings\n", len(needing), t.Label)

	fmt.Println("\n=== Generating Missing Embeddings ===")
	embedGen, err := kbembed.NewEmbeddingGenerator(config.ForProvider(t.Provider), db, maxRetries)
	if err != nil {
		return fmt.Errorf("failed to initialize embedding generator: %w", err)
	}
	embeddingErrors := embedGen.GenerateEmbeddings(needing)
	if len(embeddingErrors) > 0 {
		fmt.Println("\n⚠️  Warning: Some embedding providers failed:")
		for provider, err := range embeddingErrors {
			fmt.Printf("  - %s: %v\n", provider, err)
		}
	}

	// Embeddings are saved incrementally during generation; no final
	// update is needed.
	fmt.Printf("✓ Successfully updated %s embeddings in: %s\n", t.Label, dbPath)
	return nil
}

// missingEmbedding reports whether a chunk lacks the named provider's
// embedding.
func missingEmbedding(c *kbtypes.Chunk, provider string) bool {
	switch provider {
	case "openai":
		return len(c.OpenAIEmbedding) == 0
	case "voyage":
		return len(c.VoyageEmbedding) == 0
	case "ollama":
		return len(c.OllamaEmbedding) == 0
	case "gemini":
		return len(c.GeminiEmbedding) == 0
	}
	return false
}

func runClearEmbeddings(config *kbconfig.Config, provider string) error {
	provider = strings.ToLower(provider)
	target, ok := config.TargetForProvider(provider)
	if !ok {
		return fmt.Errorf("invalid or disabled provider %q (must be an enabled provider: openai, voyage, ollama, or gemini)", provider)
	}
	dbPath := config.DatabasePathFor(target)
	fmt.Printf("Clearing %s embeddings from: %s\n\n", provider, dbPath)

	db, err := kbdatabase.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	rowsAffected, err := db.ClearEmbeddings(provider)
	if err != nil {
		return fmt.Errorf("failed to clear embeddings: %w", err)
	}
	fmt.Printf("✓ Successfully cleared %s embeddings from %d chunks\n", provider, rowsAffected)
	return nil
}

func processAllDocuments(sources []kbsource.SourceInfo, db *kbdatabase.Database) ([]*kbtypes.Chunk, error) {
	var allChunks []*kbtypes.Chunk

	for i := range sources {
		source := &sources[i]
		fmt.Printf("\nProcessing %s %s...\n", source.Source.ProjectName, source.Source.ProjectVersion)

		chunks, err := processSource(*source, db)
		if err != nil {
			return nil, fmt.Errorf("failed to process source %s: %w", source.Source.ProjectName, err)
		}

		fmt.Printf("  Created/reused %d chunks\n", len(chunks))
		allChunks = append(allChunks, chunks...)
	}

	return allChunks, nil
}

func processSource(source kbsource.SourceInfo, db *kbdatabase.Database) ([]*kbtypes.Chunk, error) {
	var chunks []*kbtypes.Chunk
	var validChecksums []string

	// First pass: count supported files
	fmt.Printf("  Scanning for supported files...\n")
	var supportedFiles []string
	err := filepath.WalkDir(source.BasePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && kbconverter.IsSupported(path) {
			supportedFiles = append(supportedFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	fmt.Printf("  Found %d supported files\n", len(supportedFiles))

	// Second pass: process files with progress
	processedCount := 0
	for _, path := range supportedFiles {
		processedCount++

		// Show progress every file, but with relative path for readability
		relPath, err := filepath.Rel(source.BasePath, path)
		if err != nil || relPath == "" {
			relPath = filepath.Base(path)
		}

		startTime := time.Now()
		fmt.Printf("  [%d/%d] Processing: %s", processedCount, len(supportedFiles), relPath)

		// Process the file (with checksum-based incremental processing)
		fileChunks, skipped, checksum, err := processFile(path, source, db)
		elapsed := time.Since(startTime)

		if err != nil {
			fmt.Printf(" - ERROR (%.2fs): %v\n", elapsed.Seconds(), err)
			continue // Continue processing other files
		}

		// Track this checksum as valid for cleanup
		if checksum != "" {
			validChecksums = append(validChecksums, checksum)
		}

		if skipped {
			fmt.Printf(" - skipped (unchanged)\n")
		} else if len(fileChunks) > 0 {
			fmt.Printf(" - %d chunks (%.2fs)\n", len(fileChunks), elapsed.Seconds())
		} else {
			fmt.Printf(" - 0 chunks (%.2fs)\n", elapsed.Seconds())
		}

		// Add chunks to the collection
		if len(fileChunks) > 0 {
			chunks = append(chunks, fileChunks...)
		}
	}

	// Cleanup stale chunks from previous runs (files that no longer exist)
	if len(validChecksums) > 0 {
		fmt.Printf("  Cleaning up stale data...\n")
		if err := db.CleanupStaleChunks(source.Source.ProjectName, source.Source.ProjectVersion, validChecksums); err != nil {
			return nil, fmt.Errorf("failed to cleanup stale chunks: %w", err)
		}
	}

	return chunks, nil
}

func processFile(filePath string, source kbsource.SourceInfo, db *kbdatabase.Database) ([]*kbtypes.Chunk, bool, string, error) {
	stepStart := time.Now()

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false, "", fmt.Errorf("failed to read file: %w", err)
	}
	readTime := time.Since(stepStart)

	// Compute checksum
	hash := sha256.Sum256(content)
	checksum := hex.EncodeToString(hash[:])

	// Check if this file needs processing for this project/version
	needsProcessing, err := db.FileNeedsProcessing(checksum, source.Source.ProjectName, source.Source.ProjectVersion)
	if err != nil {
		return nil, false, checksum, fmt.Errorf("failed to check if file needs processing: %w", err)
	}

	// If file doesn't need processing (already processed for this project/version), skip it
	if !needsProcessing {
		// Return empty chunks - they're already in the database, no need to re-insert
		return nil, true, checksum, nil
	}

	// Check if this file exists in another version (deduplication)
	existingChunks, err := db.GetChunksForChecksum(checksum)
	if err != nil {
		return nil, false, checksum, fmt.Errorf("failed to check for existing chunks: %w", err)
	}

	// If chunks exist for this checksum in another version, clone them with new project/version
	if len(existingChunks) > 0 {
		var chunks []*kbtypes.Chunk
		for _, existingChunk := range existingChunks {
			chunk := &kbtypes.Chunk{
				Text:               existingChunk.Text,
				Title:              existingChunk.Title,
				Section:            existingChunk.Section,
				ProjectName:        source.Source.ProjectName,
				ProjectVersion:     source.Source.ProjectVersion,
				FilePath:           filePath,
				SourceFileChecksum: checksum,
				OpenAIEmbedding:    existingChunk.OpenAIEmbedding,
				VoyageEmbedding:    existingChunk.VoyageEmbedding,
				OllamaEmbedding:    existingChunk.OllamaEmbedding,
				GeminiEmbedding:    existingChunk.GeminiEmbedding,
			}
			chunks = append(chunks, chunk)
		}
		return chunks, false, checksum, nil
	}

	// File needs processing - process it from scratch
	// Detect document type
	stepStart = time.Now()
	docType := kbconverter.DetectDocumentType(filePath)
	detectTime := time.Since(stepStart)

	// Convert to markdown
	stepStart = time.Now()
	markdown, title, err := kbconverter.Convert(content, docType)
	if err != nil {
		return nil, false, checksum, fmt.Errorf("failed to convert document (read: %.2fs, detect: %.2fs, convert: %.2fs): %w",
			readTime.Seconds(), detectTime.Seconds(), time.Since(stepStart).Seconds(), err)
	}
	convertTime := time.Since(stepStart)

	// Create document
	doc := &kbtypes.Document{
		Title:          title,
		Content:        markdown,
		SourceContent:  content,
		FilePath:       filePath,
		ProjectName:    source.Source.ProjectName,
		ProjectVersion: source.Source.ProjectVersion,
		DocType:        docType,
	}

	// Chunk the document
	stepStart = time.Now()
	chunks, err := kbchunker.ChunkDocument(doc)
	if err != nil {
		return nil, false, checksum, fmt.Errorf("failed to chunk document (read: %.2fs, detect: %.2fs, convert: %.2fs, chunk: %.2fs): %w",
			readTime.Seconds(), detectTime.Seconds(), convertTime.Seconds(), time.Since(stepStart).Seconds(), err)
	}
	chunkTime := time.Since(stepStart)

	// Set checksum on all chunks
	for _, chunk := range chunks {
		chunk.SourceFileChecksum = checksum
	}

	// Log timing breakdown if file took more than 1 second
	totalTime := readTime + detectTime + convertTime + chunkTime
	if totalTime.Seconds() > 1.0 {
		fmt.Printf("           [Slow file - read: %.2fs, detect: %.2fs, convert: %.2fs, chunk: %.2fs]\n",
			readTime.Seconds(), detectTime.Seconds(), convertTime.Seconds(), chunkTime.Seconds())
	}

	return chunks, false, checksum, nil
}
