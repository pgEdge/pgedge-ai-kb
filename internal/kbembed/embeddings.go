/*-------------------------------------------------------------------------
 *
 * pgEdge AI Knowledgebase Builder
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 *-------------------------------------------------------------------------
 */

package kbembed

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pgEdge/pgedge-go-llm-lib/llm"

	"github.com/pgEdge/pgedge-ai-kb/internal/kbconfig"
	"github.com/pgEdge/pgedge-ai-kb/internal/kbdatabase"
	"github.com/pgEdge/pgedge-ai-kb/internal/kbtypes"
)

const (
	defaultBatchSize = 100
)

// EmbeddingGenerator generates embeddings using configured providers.
// All HTTP/retry/backoff concerns are delegated to pgedge-go-llm-lib;
// this type owns only chunk filtering, batching, []float32 conversion,
// and DB persistence.
type EmbeddingGenerator struct {
	clients map[string]llm.Client
	db      *kbdatabase.Database
	dbMux   sync.Mutex
}

// NewEmbeddingGenerator creates a new embedding generator. maxRetries
// controls how many times the underlying lib retries transient API
// errors: pass -1 to use the lib default, 0 for "effectively unlimited"
// (translated to a very large finite cap), or any positive integer.
func NewEmbeddingGenerator(
	config *kbconfig.Config,
	db *kbdatabase.Database,
	maxRetries int,
) (*EmbeddingGenerator, error) {
	clients := make(map[string]llm.Client)

	if config.Embeddings.OpenAI.Enabled {
		c, err := newOpenAIClient(config.Embeddings.OpenAI, maxRetries)
		if err != nil {
			return nil, fmt.Errorf("openai client: %w", err)
		}
		clients["openai"] = c
	}
	if config.Embeddings.Voyage.Enabled {
		c, err := newVoyageClient(config.Embeddings.Voyage, maxRetries)
		if err != nil {
			return nil, fmt.Errorf("voyage client: %w", err)
		}
		clients["voyage"] = c
	}
	if config.Embeddings.Gemini.Enabled {
		c, err := newGeminiClient(config.Embeddings.Gemini, maxRetries)
		if err != nil {
			return nil, fmt.Errorf("gemini client: %w", err)
		}
		clients["gemini"] = c
	}
	if config.Embeddings.Ollama.Enabled {
		c, err := newOllamaClient(config.Embeddings.Ollama, maxRetries)
		if err != nil {
			return nil, fmt.Errorf("ollama client: %w", err)
		}
		clients["ollama"] = c
	}

	return &EmbeddingGenerator{
		clients: clients,
		db:      db,
	}, nil
}

// providerSpec describes how to extract and assign a given provider's
// embedding to a chunk, and how to persist it.
type providerSpec struct {
	label   string
	get     func(*kbtypes.Chunk) []float32
	set     func(*kbtypes.Chunk, []float32)
	persist func(*kbdatabase.Database, []*kbtypes.Chunk) error
}

var providerSpecs = map[string]providerSpec{
	"openai": {
		label: "OpenAI",
		get:   func(c *kbtypes.Chunk) []float32 { return c.OpenAIEmbedding },
		set:   func(c *kbtypes.Chunk, v []float32) { c.OpenAIEmbedding = v },
		persist: func(d *kbdatabase.Database, cs []*kbtypes.Chunk) error {
			return d.UpdateOpenAIEmbeddings(cs)
		},
	},
	"voyage": {
		label: "Voyage",
		get:   func(c *kbtypes.Chunk) []float32 { return c.VoyageEmbedding },
		set:   func(c *kbtypes.Chunk, v []float32) { c.VoyageEmbedding = v },
		persist: func(d *kbdatabase.Database, cs []*kbtypes.Chunk) error {
			return d.UpdateVoyageEmbeddings(cs)
		},
	},
	"gemini": {
		label: "Gemini",
		get:   func(c *kbtypes.Chunk) []float32 { return c.GeminiEmbedding },
		set:   func(c *kbtypes.Chunk, v []float32) { c.GeminiEmbedding = v },
		persist: func(d *kbdatabase.Database, cs []*kbtypes.Chunk) error {
			return d.UpdateGeminiEmbeddings(cs)
		},
	},
	"ollama": {
		label: "Ollama",
		get:   func(c *kbtypes.Chunk) []float32 { return c.OllamaEmbedding },
		set:   func(c *kbtypes.Chunk, v []float32) { c.OllamaEmbedding = v },
		persist: func(d *kbdatabase.Database, cs []*kbtypes.Chunk) error {
			return d.UpdateOllamaEmbeddings(cs)
		},
	},
}

// GenerateEmbeddings generates embeddings for all chunks using all
// enabled providers in parallel. Returns a map of provider names to
// errors (if any); does not fail on individual provider errors.
func (eg *EmbeddingGenerator) GenerateEmbeddings(
	chunks []*kbtypes.Chunk,
) map[string]error {
	fmt.Printf("\nGenerating embeddings for %d chunks...\n", len(chunks))

	var wg sync.WaitGroup
	type providerResult struct {
		name string
		err  error
	}
	resultChan := make(chan providerResult, len(eg.clients))
	startTime := time.Now()

	for name, client := range eg.clients {
		spec, ok := providerSpecs[name]
		if !ok {
			continue
		}
		wg.Add(1)
		go func(name string, client llm.Client, spec providerSpec) {
			defer wg.Done()
			fmt.Printf("Starting %s embeddings...\n", spec.label)
			providerStart := time.Now()
			if err := eg.runProvider(context.Background(), client, spec, chunks); err != nil {
				fmt.Printf("⚠️  %s embeddings failed: %v\n", spec.label, err)
				resultChan <- providerResult{name, err}
				return
			}
			fmt.Printf("✓ %s embeddings completed in %.2fs\n",
				spec.label, time.Since(providerStart).Seconds())
			resultChan <- providerResult{name, nil}
		}(name, client, spec)
	}

	wg.Wait()
	close(resultChan)

	errs := make(map[string]error)
	for r := range resultChan {
		if r.err != nil {
			errs[r.name] = r.err
		}
	}

	fmt.Printf("\nAll embedding providers completed in %.2fs\n",
		time.Since(startTime).Seconds())
	return errs
}

// runProvider generates embeddings for one provider over all chunks
// that still need them, in batches.
func (eg *EmbeddingGenerator) runProvider(
	ctx context.Context,
	client llm.Client,
	spec providerSpec,
	chunks []*kbtypes.Chunk,
) error {
	var todo []*kbtypes.Chunk
	for _, ch := range chunks {
		if len(spec.get(ch)) == 0 && strings.TrimSpace(ch.Text) != "" {
			todo = append(todo, ch)
		}
	}

	if len(todo) == 0 {
		fmt.Printf("  %s: All chunks already have embeddings, skipping\n",
			spec.label)
		return nil
	}
	if len(todo) < len(chunks) {
		fmt.Printf(
			"  %s: Processing %d chunks (%d already have %s embeddings)\n",
			spec.label, len(todo), len(chunks)-len(todo), spec.label)
	} else {
		fmt.Printf("  %s: Processing %d chunks\n", spec.label, len(todo))
	}

	for i := 0; i < len(todo); i += defaultBatchSize {
		end := i + defaultBatchSize
		if end > len(todo) {
			end = len(todo)
		}
		batch := todo[i:end]

		texts := make([]string, len(batch))
		for j, ch := range batch {
			texts[j] = ch.Text
		}

		vecs, err := client.EmbedBatch(ctx, texts)
		if err != nil {
			if errors.Is(err, llm.ErrNotSupported) {
				return fmt.Errorf(
					"%s does not support embeddings: %w",
					spec.label, err)
			}
			return fmt.Errorf("%s embed batch %d-%d: %w",
				spec.label, i+1, end, err)
		}
		if len(vecs) != len(batch) {
			return fmt.Errorf(
				"%s: expected %d embeddings, got %d",
				spec.label, len(batch), len(vecs))
		}

		for j, ch := range batch {
			spec.set(ch, float64ToFloat32(vecs[j]))
		}

		if eg.db != nil && len(batch) > 0 && batch[0].ID != 0 {
			eg.dbMux.Lock()
			err := spec.persist(eg.db, batch)
			eg.dbMux.Unlock()
			if err != nil {
				return fmt.Errorf(
					"%s: failed to save batch to database: %w",
					spec.label, err)
			}
		}

		fmt.Printf("  %s: Processed %d/%d chunks\n",
			spec.label, end, len(todo))
	}

	return nil
}
