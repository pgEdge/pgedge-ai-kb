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
	"fmt"
	"math"
	"time"

	"github.com/pgEdge/pgedge-go-llm-lib/llm"
	_ "github.com/pgEdge/pgedge-go-llm-lib/llm/provider/gemini"
	"github.com/pgEdge/pgedge-go-llm-lib/llm/provider/ollama"
	"github.com/pgEdge/pgedge-go-llm-lib/llm/provider/openai"
	_ "github.com/pgEdge/pgedge-go-llm-lib/llm/provider/voyage"

	"github.com/pgEdge/pgedge-ai-kb/internal/kbconfig"
)

// libRetry maps the kb-builder "0 means unlimited" convention onto a
// finite cap acceptable to the LLM library.
func libRetry(maxRetries int) llm.RetryConfig {
	if maxRetries == 0 {
		// "Unlimited" in kb-builder. The lib requires a positive
		// integer; use a very large finite cap.
		maxRetries = math.MaxInt32
	}
	if maxRetries < 0 {
		maxRetries = 5
	}
	return llm.RetryConfig{
		MaxRetries:     maxRetries,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     60 * time.Second,
	}
}

// timeouts carries the embedding request timeouts shared by every
// provider client. RequestTimeout is the overall ceiling for one
// request including retries; PerAttemptTimeout bounds each HTTP attempt
// so a stalled attempt (notably Gemini's heavy batchEmbedContents) is
// retried rather than cancelling the whole request.
type timeouts struct {
	request    time.Duration
	perAttempt time.Duration
}

// onRetryLogger returns an OnRetry hook that mimics the kb-builder's
// historical "Retry N for X after Ys..." log lines.
func onRetryLogger(label string) func(llm.RetryEvent) {
	return func(e llm.RetryEvent) {
		if e.StatusCode == 429 {
			fmt.Printf("  Rate limited (%s), retrying in %.1fs...\n",
				label, e.Wait.Seconds())
			return
		}
		fmt.Printf("  Retry %d for %s after %.1fs...\n",
			e.Attempt, label, e.Wait.Seconds())
	}
}

// newOpenAIClient builds an llm.Client configured for OpenAI embeddings.
func newOpenAIClient(cfg kbconfig.OpenAIConfig, maxRetries int, to timeouts) (llm.Client, error) {
	return llm.NewClient("openai", llm.Options{
		APIKey:            cfg.APIKey,
		Model:             cfg.Model,
		Retry:             libRetry(maxRetries),
		RequestTimeout:    to.request,
		PerAttemptTimeout: to.perAttempt,
		Extensions: []llm.ProviderExtension{
			openai.Extension{EmbeddingDimensions: cfg.Dimensions},
		},
		OnRetry: onRetryLogger("OpenAI"),
	})
}

// newVoyageClient builds an llm.Client configured for Voyage embeddings.
func newVoyageClient(cfg kbconfig.VoyageConfig, maxRetries int, to timeouts) (llm.Client, error) {
	return llm.NewClient("voyage", llm.Options{
		APIKey:            cfg.APIKey,
		Model:             cfg.Model,
		Retry:             libRetry(maxRetries),
		RequestTimeout:    to.request,
		PerAttemptTimeout: to.perAttempt,
		OnRetry:           onRetryLogger("Voyage"),
	})
}

// newGeminiClient builds an llm.Client configured for Gemini embeddings.
func newGeminiClient(cfg kbconfig.GeminiConfig, maxRetries int, to timeouts) (llm.Client, error) {
	return llm.NewClient("gemini", llm.Options{
		APIKey:            cfg.APIKey,
		Model:             cfg.Model,
		Retry:             libRetry(maxRetries),
		RequestTimeout:    to.request,
		PerAttemptTimeout: to.perAttempt,
		OnRetry:           onRetryLogger("Gemini"),
	})
}

// newOllamaClient builds an llm.Client configured for Ollama embeddings,
// with Authorization injected for Ollama Cloud when an API key is set.
func newOllamaClient(cfg kbconfig.OllamaConfig, maxRetries int, to timeouts) (llm.Client, error) {
	opts := llm.Options{
		BaseURL:           cfg.Endpoint,
		Model:             cfg.Model,
		Retry:             libRetry(maxRetries),
		RequestTimeout:    to.request,
		PerAttemptTimeout: to.perAttempt,
		Extensions: []llm.ProviderExtension{
			ollama.Extension{
				EmbedContextLength:      cfg.ContextLength,
				EmbedTruncateOnOverflow: true,
			},
		},
		OnRetry: onRetryLogger("Ollama"),
	}
	if cfg.APIKey != "" {
		opts.CustomHeaders = map[string]string{
			"Authorization": "Bearer " + cfg.APIKey,
		}
	}
	return llm.NewClient("ollama", opts)
}

// float64ToFloat32 converts an embedding returned by the LLM library
// to the kb-builder's storage format.
func float64ToFloat32(in []float64) []float32 {
	if in == nil {
		return nil
	}
	out := make([]float32, len(in))
	for i, v := range in {
		out[i] = float32(v)
	}
	return out
}
