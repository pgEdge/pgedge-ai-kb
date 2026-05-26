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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pgEdge/pgedge-go-llm-lib/llm"
	_ "github.com/pgEdge/pgedge-go-llm-lib/llm/provider/openai"

	"github.com/pgEdge/pgedge-ai-kb/internal/kbconfig"
	"github.com/pgEdge/pgedge-ai-kb/internal/kbtypes"
)

// openaiMockServer returns an httptest.Server that mimics the OpenAI
// embeddings endpoint and emits dummy float vectors.
func openaiMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter, r *http.Request,
	) {
		var body struct {
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		type item struct {
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		}
		out := struct {
			Data []item `json:"data"`
		}{}
		for i := range body.Input {
			out.Data = append(out.Data, item{
				Embedding: []float64{0.1, 0.2, 0.3},
				Index:     i,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}))
}

// mustOpenAIClientWithBaseURL builds an OpenAI llm.Client whose
// requests are routed to baseURL (the test server).
func mustOpenAIClientWithBaseURL(t *testing.T, baseURL string) llm.Client {
	t.Helper()
	c, err := llm.NewClient("openai", llm.Options{
		APIKey:  "test-key",
		Model:   "text-embedding-3-small",
		BaseURL: baseURL,
	})
	if err != nil {
		t.Fatalf("llm.NewClient: %v", err)
	}
	return c
}

func TestNewEmbeddingGenerator_OpenAI(t *testing.T) {
	cfg := &kbconfig.Config{
		Embeddings: kbconfig.EmbeddingConfig{
			OpenAI: kbconfig.OpenAIConfig{
				Enabled: true,
				APIKey:  "test-key",
				Model:   "text-embedding-3-small",
			},
		},
	}

	eg, err := NewEmbeddingGenerator(cfg, nil, -1)
	if err != nil {
		t.Fatalf("NewEmbeddingGenerator: %v", err)
	}
	if _, ok := eg.clients["openai"]; !ok {
		t.Error("expected openai client to be registered")
	}
}

func TestGenerateEmbeddings_NoProvidersEnabled(t *testing.T) {
	cfg := &kbconfig.Config{
		Embeddings: kbconfig.EmbeddingConfig{},
	}
	eg, err := NewEmbeddingGenerator(cfg, nil, -1)
	if err != nil {
		t.Fatalf("NewEmbeddingGenerator: %v", err)
	}
	errs := eg.GenerateEmbeddings([]*kbtypes.Chunk{
		{Text: "x", ProjectName: "P", ProjectVersion: "1"},
	})
	if len(errs) != 0 {
		t.Errorf("expected no errors with no providers enabled, got: %v", errs)
	}
}

// TestGenerateEmbeddings_OpenAI_EndToEnd exercises the full code path
// (client construction → EmbedBatch → []float32 assignment) against an
// httptest.Server pretending to be OpenAI.
func TestGenerateEmbeddings_OpenAI_EndToEnd(t *testing.T) {
	srv := openaiMockServer(t)
	defer srv.Close()

	cfg := &kbconfig.Config{
		Embeddings: kbconfig.EmbeddingConfig{
			OpenAI: kbconfig.OpenAIConfig{
				Enabled: true,
				APIKey:  "test-key",
				Model:   "text-embedding-3-small",
			},
		},
	}
	eg, err := NewEmbeddingGenerator(cfg, nil, -1)
	if err != nil {
		t.Fatalf("NewEmbeddingGenerator: %v", err)
	}
	// Replace the OpenAI client with one pointed at the mock server.
	eg.clients["openai"] = mustOpenAIClientWithBaseURL(t, srv.URL)

	chunks := []*kbtypes.Chunk{
		{Text: "hello", ProjectName: "P", ProjectVersion: "1"},
		{Text: "world", ProjectName: "P", ProjectVersion: "1"},
	}
	if errs := eg.GenerateEmbeddings(chunks); len(errs) != 0 {
		t.Fatalf("GenerateEmbeddings errors: %v", errs)
	}
	for i, ch := range chunks {
		if len(ch.OpenAIEmbedding) != 3 {
			t.Errorf("chunk %d: OpenAIEmbedding len = %d, want 3",
				i, len(ch.OpenAIEmbedding))
		}
	}
}

func TestGenerateEmbeddings_EmptyChunks(t *testing.T) {
	srv := openaiMockServer(t)
	defer srv.Close()

	cfg := &kbconfig.Config{
		Embeddings: kbconfig.EmbeddingConfig{
			OpenAI: kbconfig.OpenAIConfig{
				Enabled: true,
				APIKey:  "test-key",
				Model:   "text-embedding-3-small",
			},
		},
	}
	eg, err := NewEmbeddingGenerator(cfg, nil, -1)
	if err != nil {
		t.Fatalf("NewEmbeddingGenerator: %v", err)
	}
	eg.clients["openai"] = mustOpenAIClientWithBaseURL(t, srv.URL)

	if errs := eg.GenerateEmbeddings([]*kbtypes.Chunk{}); len(errs) != 0 {
		t.Errorf("expected no errors for empty chunks, got: %v", errs)
	}
}

func TestFloat64ToFloat32(t *testing.T) {
	in := []float64{0.1, 0.2, 0.3}
	out := float64ToFloat32(in)
	if len(out) != len(in) {
		t.Fatalf("len mismatch: %d vs %d", len(out), len(in))
	}
	for i, v := range out {
		want := float32(in[i])
		if v != want {
			t.Errorf("out[%d] = %v, want %v", i, v, want)
		}
	}
	if float64ToFloat32(nil) != nil {
		t.Error("nil input should produce nil output")
	}
}
