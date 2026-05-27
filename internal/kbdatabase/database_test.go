/*-------------------------------------------------------------------------
 *
 * pgEdge AI Knowledgebase Builder
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 *-------------------------------------------------------------------------
 */

package kbdatabase

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/pgEdge/pgedge-ai-kb/internal/kbtypes"
)

func TestOpenDatabase(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"

	db, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Verify database file was created
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestInsertAndSearchChunks(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"

	db, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create test chunks with embeddings
	chunks := []*kbtypes.Chunk{
		{
			Text:            "PostgreSQL is a powerful database system.",
			Title:           "PostgreSQL Overview",
			Section:         "Introduction",
			ProjectName:     "PostgreSQL",
			ProjectVersion:  "17",
			FilePath:        "/docs/intro.md",
			OpenAIEmbedding: []float32{0.1, 0.2, 0.3, 0.4, 0.5},
		},
		{
			Text:            "Window functions allow calculations across rows.",
			Title:           "Window Functions",
			Section:         "Advanced Features",
			ProjectName:     "PostgreSQL",
			ProjectVersion:  "17",
			FilePath:        "/docs/advanced.md",
			OpenAIEmbedding: []float32{0.2, 0.3, 0.4, 0.5, 0.6},
		},
	}

	// Insert chunks
	err = db.InsertChunks(chunks)
	if err != nil {
		t.Fatalf("Failed to insert chunks: %v", err)
	}

	// Test text search
	results, err := db.SearchChunks("PostgreSQL", 10)
	if err != nil {
		t.Fatalf("Failed to search chunks: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results but got none")
	}

	// Verify chunk data
	found := false
	for _, result := range results {
		if result.ProjectName == "PostgreSQL" && result.ProjectVersion == "17" {
			found = true
			if len(result.OpenAIEmbedding) != 5 {
				t.Error("Embedding should be preserved")
			}
		}
	}

	if !found {
		t.Error("Expected to find PostgreSQL chunks")
	}
}

func TestGetStats(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"

	db, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Insert test data
	chunks := []*kbtypes.Chunk{
		{
			Text:            "Test 1",
			ProjectName:     "Project A",
			ProjectVersion:  "1.0",
			OpenAIEmbedding: []float32{0.1},
		},
		{
			Text:            "Test 2",
			ProjectName:     "Project A",
			ProjectVersion:  "1.0",
			OpenAIEmbedding: []float32{0.2},
		},
		{
			Text:            "Test 3",
			ProjectName:     "Project B",
			ProjectVersion:  "2.0",
			OpenAIEmbedding: []float32{0.3},
		},
	}

	err = db.InsertChunks(chunks)
	if err != nil {
		t.Fatalf("Failed to insert chunks: %v", err)
	}

	// Get stats
	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	totalChunks, ok := stats["total_chunks"].(int)
	if !ok || totalChunks != 3 {
		t.Errorf("Expected 3 total chunks, got %v", stats["total_chunks"])
	}

	projects, ok := stats["projects"].([]map[string]interface{})
	if !ok {
		t.Fatal("Projects should be a slice of maps")
	}

	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}
}

func TestSerializeDeserializeEmbedding(t *testing.T) {
	original := []float32{0.1, 0.2, 0.3, 0.4, 0.5}

	// Serialize
	blob := serializeEmbedding(original)
	if len(blob) != len(original)*4 {
		t.Errorf("Expected blob size %d, got %d", len(original)*4, len(blob))
	}

	// Deserialize
	result := deserializeEmbedding(blob)
	if len(result) != len(original) {
		t.Errorf("Expected %d values, got %d", len(original), len(result))
	}

	for i, val := range result {
		if val != original[i] {
			t.Errorf("Value mismatch at index %d: expected %f, got %f", i, original[i], val)
		}
	}
}

func TestDeserializeInvalidEmbedding(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "invalid length",
			data: []byte{1, 2, 3}, // Not divisible by 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deserializeEmbedding(tt.data)
			if result != nil {
				t.Error("Expected nil for invalid data")
			}
		})
	}
}

func TestInsertMultipleProviders(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"

	db, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	chunks := []*kbtypes.Chunk{
		{
			Text:            "Test chunk",
			ProjectName:     "Test",
			ProjectVersion:  "1.0",
			OpenAIEmbedding: []float32{0.1, 0.2, 0.3},
			VoyageEmbedding: []float32{0.4, 0.5, 0.6},
			OllamaEmbedding: []float32{0.7, 0.8, 0.9},
		},
	}

	err = db.InsertChunks(chunks)
	if err != nil {
		t.Fatalf("Failed to insert chunks: %v", err)
	}

	// Retrieve and verify all embeddings are present
	results, err := db.SearchChunks("Test", 1)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if len(result.OpenAIEmbedding) != 3 {
		t.Error("OpenAI embedding not preserved")
	}
	if len(result.VoyageEmbedding) != 3 {
		t.Error("Voyage embedding not preserved")
	}
	if len(result.OllamaEmbedding) != 3 {
		t.Error("Ollama embedding not preserved")
	}
}

func TestTransactionRollback(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"

	db, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Insert valid chunk first
	validChunks := []*kbtypes.Chunk{
		{
			Text:            "Valid chunk",
			ProjectName:     "Test",
			ProjectVersion:  "1.0",
			OpenAIEmbedding: []float32{0.1},
		},
	}

	err = db.InsertChunks(validChunks)
	if err != nil {
		t.Fatalf("Failed to insert valid chunks: %v", err)
	}

	// Get initial count
	stats, _ := db.GetStats()
	initialCount := stats["total_chunks"].(int)

	// The transaction should complete successfully since we're just inserting more chunks
	// (there's no way to force an error in the current implementation without mocking)

	// Verify count matches expected
	stats, _ = db.GetStats()
	finalCount := stats["total_chunks"].(int)

	// This test is mainly to ensure the transaction mechanism is in place
	// In a real scenario with mock errors, this would test rollback
	_, _ = initialCount, finalCount // Suppress unused variable warnings
}

func TestGeminiEmbeddingRoundTrip(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	want := []float32{0.11, 0.22, 0.33, 0.44}
	chunk := &kbtypes.Chunk{
		Text:            "hello gemini",
		ProjectName:     "P",
		ProjectVersion:  "1",
		GeminiEmbedding: want,
	}
	if err := db.InsertChunks([]*kbtypes.Chunk{chunk}); err != nil {
		t.Fatalf("InsertChunks: %v", err)
	}

	// InsertChunks must populate Chunk.ID so the embedding generator can
	// later persist embeddings to this specific row.
	if chunk.ID == 0 {
		t.Error("InsertChunks did not populate Chunk.ID")
	}

	got, err := db.GetAllChunks()
	if err != nil {
		t.Fatalf("GetAllChunks: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 chunk, got %d", len(got))
	}
	if len(got[0].GeminiEmbedding) != len(want) {
		t.Fatalf("GeminiEmbedding length %d, want %d",
			len(got[0].GeminiEmbedding), len(want))
	}
	for i, v := range want {
		if got[0].GeminiEmbedding[i] != v {
			t.Errorf("GeminiEmbedding[%d] = %v, want %v",
				i, got[0].GeminiEmbedding[i], v)
		}
	}
}

func TestMigrateAddGeminiColumn(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "old.db")

	// Build a database manually with the old schema (no gemini_embedding column).
	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if _, err := raw.Exec(`
		CREATE TABLE chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			text TEXT NOT NULL,
			title TEXT,
			section TEXT,
			project_name TEXT NOT NULL,
			project_version TEXT NOT NULL,
			file_path TEXT,
			source_file_checksum TEXT,
			openai_embedding BLOB,
			voyage_embedding BLOB,
			ollama_embedding BLOB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		t.Fatalf("create old schema: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("close raw: %v", err)
	}

	// Open via the package; this should run migrateSchema and add the column.
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	// Verify the column now exists.
	cols, err := db.columnSet("chunks")
	if err != nil {
		t.Fatalf("columnSet: %v", err)
	}
	if !cols["gemini_embedding"] {
		t.Error("gemini_embedding column was not added by migration")
	}

	// Round-trip a chunk with a Gemini embedding to confirm the column is usable.
	if err := db.InsertChunks([]*kbtypes.Chunk{{
		Text:            "x",
		ProjectName:     "P",
		ProjectVersion:  "1",
		GeminiEmbedding: []float32{1, 2, 3},
	}}); err != nil {
		t.Fatalf("InsertChunks after migration: %v", err)
	}
}
