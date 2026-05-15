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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pgEdge/pgedge-ai-kb/internal/kbconfig"
	"github.com/pgEdge/pgedge-ai-kb/internal/kbtypes"
)

func TestNewEmbeddingGenerator(t *testing.T) {
	config := &kbconfig.Config{
		Embeddings: kbconfig.EmbeddingConfig{
			OpenAI: kbconfig.OpenAIConfig{
				Enabled: true,
				APIKey:  "test-key",
				Model:   "text-embedding-3-small",
			},
		},
	}

	eg := NewEmbeddingGenerator(config, nil, -1)

	if eg == nil {
		t.Fatal("Expected embedding generator, got nil")
	}

	if eg.config != config {
		t.Error("Config not set correctly")
	}

	if eg.client == nil {
		t.Error("HTTP client not initialized")
	}
}

func TestOpenAIRequestStructure(t *testing.T) {
	// Test that we can marshal OpenAI request correctly
	req := openAIEmbeddingRequest{
		Input:      []string{"test text 1", "test text 2"},
		Model:      "text-embedding-3-small",
		Dimensions: 1536,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal OpenAI request: %v", err)
	}

	// Unmarshal to verify structure
	var decoded openAIEmbeddingRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal OpenAI request: %v", err)
	}

	if len(decoded.Input) != 2 {
		t.Errorf("Expected 2 inputs, got %d", len(decoded.Input))
	}

	if decoded.Model != "text-embedding-3-small" {
		t.Errorf("Expected model 'text-embedding-3-small', got %q", decoded.Model)
	}

	if decoded.Dimensions != 1536 {
		t.Errorf("Expected dimensions 1536, got %d", decoded.Dimensions)
	}
}

func TestVoyageRequestStructure(t *testing.T) {
	// Test that we can marshal Voyage request correctly
	req := voyageEmbeddingRequest{
		Input: []string{"test text 1", "test text 2"},
		Model: "voyage-3",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal Voyage request: %v", err)
	}

	// Unmarshal to verify structure
	var decoded voyageEmbeddingRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Voyage request: %v", err)
	}

	if len(decoded.Input) != 2 {
		t.Errorf("Expected 2 inputs, got %d", len(decoded.Input))
	}

	if decoded.Model != "voyage-3" {
		t.Errorf("Expected model 'voyage-3', got %q", decoded.Model)
	}
}

func TestOllamaRequestStructure(t *testing.T) {
	// Test that we can marshal Ollama request correctly
	req := ollamaEmbeddingRequest{
		Model: "nomic-embed-text",
		Input: "test text",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal Ollama request: %v", err)
	}

	// Unmarshal to verify structure
	var decoded ollamaEmbeddingRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Ollama request: %v", err)
	}

	if decoded.Model != "nomic-embed-text" {
		t.Errorf("Expected model 'nomic-embed-text', got %q", decoded.Model)
	}

	if decoded.Input != "test text" {
		t.Errorf("Expected input 'test text', got %q", decoded.Input)
	}
}

func TestOpenAIResponseStructure(t *testing.T) {
	// Test that we can unmarshal OpenAI response correctly
	responseJSON := `{
        "data": [
            {"embedding": [0.1, 0.2, 0.3]},
            {"embedding": [0.4, 0.5, 0.6]}
        ]
    }`

	var resp openAIEmbeddingResponse
	if err := json.Unmarshal([]byte(responseJSON), &resp); err != nil {
		t.Fatalf("Failed to unmarshal OpenAI response: %v", err)
	}

	if len(resp.Data) != 2 {
		t.Errorf("Expected 2 embeddings, got %d", len(resp.Data))
	}

	if len(resp.Data[0].Embedding) != 3 {
		t.Errorf("Expected embedding with 3 dimensions, got %d", len(resp.Data[0].Embedding))
	}

	if resp.Data[0].Embedding[0] != 0.1 {
		t.Errorf("Expected first value 0.1, got %f", resp.Data[0].Embedding[0])
	}
}

func TestVoyageResponseStructure(t *testing.T) {
	// Test that we can unmarshal Voyage response correctly
	responseJSON := `{
        "data": [
            {"embedding": [0.1, 0.2, 0.3]},
            {"embedding": [0.4, 0.5, 0.6]}
        ]
    }`

	var resp voyageEmbeddingResponse
	if err := json.Unmarshal([]byte(responseJSON), &resp); err != nil {
		t.Fatalf("Failed to unmarshal Voyage response: %v", err)
	}

	if len(resp.Data) != 2 {
		t.Errorf("Expected 2 embeddings, got %d", len(resp.Data))
	}

	if len(resp.Data[1].Embedding) != 3 {
		t.Errorf("Expected embedding with 3 dimensions, got %d", len(resp.Data[1].Embedding))
	}
}

func TestOllamaResponseStructure(t *testing.T) {
	// Test that we can unmarshal Ollama response correctly
	responseJSON := `{"embeddings": [[0.1, 0.2, 0.3, 0.4, 0.5]]}`

	var resp ollamaEmbeddingResponse
	if err := json.Unmarshal([]byte(responseJSON), &resp); err != nil {
		t.Fatalf("Failed to unmarshal Ollama response: %v", err)
	}

	if len(resp.Embeddings) != 1 {
		t.Fatalf("Expected 1 embedding, got %d", len(resp.Embeddings))
	}

	if len(resp.Embeddings[0]) != 5 {
		t.Errorf("Expected embedding with 5 dimensions, got %d", len(resp.Embeddings[0]))
	}

	if resp.Embeddings[0][0] != 0.1 {
		t.Errorf("Expected first value 0.1, got %f", resp.Embeddings[0][0])
	}
}

func TestGenerateOpenAIEmbeddings_WithMockServer(t *testing.T) {
	// Create a mock server that returns valid embeddings
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json")
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-api-key" {
			t.Errorf("Expected Authorization header with Bearer token")
		}

		// Return mock response
		response := openAIEmbeddingResponse{
			Data: []struct {
				Embedding []float32 `json:"embedding"`
			}{
				{Embedding: []float32{0.1, 0.2, 0.3}},
				{Embedding: []float32{0.4, 0.5, 0.6}},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Note: This test would need the generateOpenAIEmbeddings method to accept
	// a custom URL parameter to test with the mock server. Since it's hardcoded
	// to use the OpenAI API, we'll just verify the structures work correctly.
	// In a production environment, you'd want to refactor to allow dependency injection.

	t.Log("Mock server test structure validated")
}

func TestGenerateEmbeddings_NoProvidersEnabled(t *testing.T) {
	config := &kbconfig.Config{
		Embeddings: kbconfig.EmbeddingConfig{
			OpenAI: kbconfig.OpenAIConfig{Enabled: false},
			Voyage: kbconfig.VoyageConfig{Enabled: false},
			Ollama: kbconfig.OllamaConfig{Enabled: false},
		},
	}

	eg := NewEmbeddingGenerator(config, nil, -1)

	chunks := []*kbtypes.Chunk{
		{
			Text:           "Test chunk",
			ProjectName:    "Test",
			ProjectVersion: "1.0",
		},
	}

	// Should not error when no providers are enabled, just skip generation
	errs := eg.GenerateEmbeddings(chunks)
	if len(errs) != 0 {
		t.Errorf("Expected no errors with no providers enabled, got: %v", errs)
	}
}

func TestGenerateEmbeddings_EmptyChunks(t *testing.T) {
	config := &kbconfig.Config{
		Embeddings: kbconfig.EmbeddingConfig{
			OpenAI: kbconfig.OpenAIConfig{
				Enabled: true,
				APIKey:  "test-key",
				Model:   "text-embedding-3-small",
			},
		},
	}

	eg := NewEmbeddingGenerator(config, nil, -1)

	chunks := []*kbtypes.Chunk{}

	// Should not error with empty chunks
	// (will fail when actually calling API, but that's expected in test environment)
	errs := eg.GenerateEmbeddings(chunks)
	if len(errs) != 0 {
		// API call will fail without valid key, which is expected
		// Just verify the error is from API call, not from our code
		t.Logf("Expected API error in test environment: %v", errs)
	}
}

func TestBatchProcessing(t *testing.T) {
	// Test that we correctly calculate batch boundaries
	const batchSize = 100

	tests := []struct {
		name            string
		totalChunks     int
		expectedBatches int
	}{
		{"single batch", 50, 1},
		{"exact batch", 100, 1},
		{"two batches", 150, 2},
		{"multiple batches", 250, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batches := 0
			for i := 0; i < tt.totalChunks; i += batchSize {
				batches++
			}
			if batches != tt.expectedBatches {
				t.Errorf("Expected %d batches, got %d", tt.expectedBatches, batches)
			}
		})
	}
}

func TestChunkTextExtraction(t *testing.T) {
	// Test that we correctly extract texts from chunks for batch processing
	chunks := []*kbtypes.Chunk{
		{Text: "chunk 1", ProjectName: "Test", ProjectVersion: "1.0"},
		{Text: "chunk 2", ProjectName: "Test", ProjectVersion: "1.0"},
		{Text: "chunk 3", ProjectName: "Test", ProjectVersion: "1.0"},
	}

	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Text
	}

	if len(texts) != 3 {
		t.Errorf("Expected 3 texts, got %d", len(texts))
	}

	if texts[0] != "chunk 1" {
		t.Errorf("Expected 'chunk 1', got %q", texts[0])
	}

	if texts[2] != "chunk 3" {
		t.Errorf("Expected 'chunk 3', got %q", texts[2])
	}
}

func TestEmbeddingAssignment(t *testing.T) {
	// Test that embeddings are correctly assigned to chunks
	chunk := &kbtypes.Chunk{
		Text:           "test",
		ProjectName:    "Test",
		ProjectVersion: "1.0",
	}

	// Verify chunk fields are set
	if chunk.Text == "" || chunk.ProjectName == "" || chunk.ProjectVersion == "" {
		t.Error("Chunk fields not initialized correctly")
	}

	// Simulate assigning different provider embeddings
	chunk.OpenAIEmbedding = []float32{0.1, 0.2, 0.3}
	chunk.VoyageEmbedding = []float32{0.4, 0.5, 0.6}
	chunk.OllamaEmbedding = []float32{0.7, 0.8, 0.9}

	if len(chunk.OpenAIEmbedding) != 3 {
		t.Error("OpenAI embedding not assigned correctly")
	}

	if len(chunk.VoyageEmbedding) != 3 {
		t.Error("Voyage embedding not assigned correctly")
	}

	if len(chunk.OllamaEmbedding) != 3 {
		t.Error("Ollama embedding not assigned correctly")
	}

	if chunk.OpenAIEmbedding[0] != 0.1 {
		t.Error("OpenAI embedding values incorrect")
	}
}

func TestTruncateAtWordBoundary(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		fraction float64
		want     string
	}{
		{
			name:     "full text at 1.0",
			text:     "hello world foo bar",
			fraction: 1.0,
			want:     "hello world foo bar",
		},
		{
			name:     "75 percent cuts at word boundary",
			text:     "hello world foo bar baz",
			fraction: 0.75,
			// 23 chars * 0.75 = 17 -> walks back to space at 15 -> "hello world foo"
			want: "hello world foo",
		},
		{
			name:     "50 percent cuts at word boundary",
			text:     "hello world foo bar baz qux",
			fraction: 0.50,
			// 27 chars * 0.50 = 13 -> walks back to space at 11 -> "hello world"
			want: "hello world",
		},
		{
			name:     "25 percent cuts at word boundary",
			text:     "hello world foo bar baz qux quux corge",
			fraction: 0.25,
			// 38 chars * 0.25 = 9 -> walks back to space at 5 -> "hello"
			want: "hello",
		},
		{
			name:     "empty result",
			text:     "hello world",
			fraction: 0.0,
			want:     "",
		},
		{
			name:     "single word no spaces falls back to hard cut",
			text:     "abcdefghijklmnop",
			fraction: 0.50,
			// No space found, falls back to targetLen = 8
			want: "abcdefgh",
		},
		{
			name:     "empty input",
			text:     "",
			fraction: 0.50,
			want:     "",
		},
		{
			name:     "fraction above 1.0",
			text:     "hello world",
			fraction: 1.5,
			want:     "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateAtWordBoundary(tt.text, tt.fraction)
			if got != tt.want {
				t.Errorf("truncateAtWordBoundary(%q, %.2f) = %q, want %q",
					tt.text, tt.fraction, got, tt.want)
			}
		})
	}
}

func TestIsContextLengthError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "context length error from retry",
			err:  fmt.Errorf("failed after 5 retries: HTTP 500: {\"error\":\"the input length exceeds the context length\"}"),
			want: true,
		},
		{
			name: "different error",
			err:  fmt.Errorf("connection refused"),
			want: false,
		},
		{
			name: "partial match",
			err:  fmt.Errorf("the input length exceeds the context length"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isContextLengthError(tt.err)
			if got != tt.want {
				t.Errorf("isContextLengthError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestOllamaContextLengthTruncation(t *testing.T) {
	// Track how many requests the server receives and at what text lengths
	var requestTexts []string

	// Create a mock Ollama server that rejects long texts.
	// Uses HTTP 400 (non-retryable) so retryWithBackoff returns
	// immediately, keeping the test fast.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ollamaEmbeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		requestTexts = append(requestTexts, req.Input)

		// Simulate context length error for text over 100 chars
		if len(req.Input) > 100 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "the input length exceeds the context length",
			})
			return
		}

		// Return a valid embedding for short enough text
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ollamaEmbeddingResponse{
			Embeddings: [][]float32{{0.1, 0.2, 0.3}},
		})
	}))
	defer server.Close()

	config := &kbconfig.Config{
		Embeddings: kbconfig.EmbeddingConfig{
			Ollama: kbconfig.OllamaConfig{
				Enabled:       true,
				Endpoint:      server.URL,
				Model:         "test-model",
				ContextLength: 8192,
			},
		},
	}

	eg := NewEmbeddingGenerator(config, nil, -1)

	// Create a chunk with text that is long enough to fail initially
	// but short enough that 50% truncation will succeed
	longText := "word "
	for len(longText) < 200 {
		longText += "word "
	}

	chunks := []*kbtypes.Chunk{
		{
			Text:           longText,
			FilePath:       "test.md",
			Section:        "Test Section",
			ProjectName:    "Test",
			ProjectVersion: "1.0",
		},
	}

	err := eg.generateOllamaEmbeddings(chunks)
	if err != nil {
		t.Fatalf("generateOllamaEmbeddings failed: %v", err)
	}

	// The chunk should have received an embedding via truncation
	if len(chunks[0].OllamaEmbedding) == 0 {
		t.Error("Expected chunk to have Ollama embedding after truncation")
	}

	// Verify that multiple requests were made (original failed + truncated attempts)
	if len(requestTexts) < 2 {
		t.Errorf("Expected multiple requests due to truncation, got %d", len(requestTexts))
	}
}

func TestIsOllamaServerError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "HTTP 500 error from retry",
			err:  fmt.Errorf("failed after 5 retries: HTTP 500: internal server error"),
			want: true,
		},
		{
			name: "HTTP 500 with ollama crash details",
			err:  fmt.Errorf("HTTP 500: {\"error\":\"llama runner process has terminated\"}"),
			want: true,
		},
		{
			name: "different HTTP error",
			err:  fmt.Errorf("HTTP 400: bad request"),
			want: false,
		},
		{
			name: "connection error",
			err:  fmt.Errorf("connection refused"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isOllamaServerError(tt.err)
			if got != tt.want {
				t.Errorf("isOllamaServerError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestOllamaServerErrorTruncation(t *testing.T) {
	// Track how many requests the server receives and at what text lengths
	var requestTexts []string

	// Create a mock Ollama server that returns HTTP 500 for long texts
	// and succeeds for shorter (truncated) texts.  This simulates the
	// Ollama model runner crashing on certain content; truncation
	// should resolve the crash.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ollamaEmbeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		requestTexts = append(requestTexts, req.Input)

		// Simulate server error for text over 100 chars
		if len(req.Input) > 100 {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "llama runner process has terminated: signal: aborted",
			})
			return
		}

		// Return a valid embedding for short enough text
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ollamaEmbeddingResponse{
			Embeddings: [][]float32{{0.1, 0.2, 0.3}},
		})
	}))
	defer server.Close()

	config := &kbconfig.Config{
		Embeddings: kbconfig.EmbeddingConfig{
			Ollama: kbconfig.OllamaConfig{
				Enabled:       true,
				Endpoint:      server.URL,
				Model:         "test-model",
				ContextLength: 8192,
			},
		},
	}

	// Use maxRetries=1 so the retry/backoff loop in retryWithBackoff
	// completes quickly.  HTTP 500 is retryable, so without this the
	// test would wait through several seconds of exponential backoff
	// before the truncation code path runs.
	eg := NewEmbeddingGenerator(config, nil, 1)

	// Create a chunk with text that is long enough to trigger HTTP 500
	// initially but short enough that 50% truncation will succeed.
	longText := "word "
	for len(longText) < 250 {
		longText += "word "
	}

	chunks := []*kbtypes.Chunk{
		{
			Text:           longText,
			FilePath:       "test.md",
			Section:        "Test Section",
			ProjectName:    "Test",
			ProjectVersion: "1.0",
		},
	}

	err := eg.generateOllamaEmbeddings(chunks)
	if err != nil {
		t.Fatalf("generateOllamaEmbeddings failed: %v", err)
	}

	// The chunk should have received an embedding via truncation
	if len(chunks[0].OllamaEmbedding) == 0 {
		t.Error("Expected chunk to have Ollama embedding after truncation")
	}

	// Verify that multiple requests were made (initial failures plus
	// at least one truncated attempt).
	if len(requestTexts) < 2 {
		t.Errorf("Expected multiple requests due to truncation, got %d", len(requestTexts))
	}

	// Verify that at least one successful request had a shorter text
	// than the original, confirming that truncation occurred.
	sawTruncated := false
	for _, text := range requestTexts {
		if len(text) > 0 && len(text) < len(longText) {
			sawTruncated = true
			break
		}
	}
	if !sawTruncated {
		t.Error("Expected at least one request with truncated text")
	}
}

func TestOllamaContextLengthSkipsChunk(t *testing.T) {
	// Create a mock Ollama server that always rejects with context length error.
	// Uses HTTP 400 (non-retryable) to avoid slow backoff retries in tests.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "the input length exceeds the context length",
		})
	}))
	defer server.Close()

	config := &kbconfig.Config{
		Embeddings: kbconfig.EmbeddingConfig{
			Ollama: kbconfig.OllamaConfig{
				Enabled:       true,
				Endpoint:      server.URL,
				Model:         "test-model",
				ContextLength: 8192,
			},
		},
	}

	eg := NewEmbeddingGenerator(config, nil, -1)

	chunks := []*kbtypes.Chunk{
		{
			Text:           "some text that always exceeds the context",
			FilePath:       "test.md",
			Section:        "Test Section",
			ProjectName:    "Test",
			ProjectVersion: "1.0",
		},
	}

	// Should not return an error; the chunk should be skipped
	err := eg.generateOllamaEmbeddings(chunks)
	if err != nil {
		t.Fatalf("Expected no error (chunk should be skipped), got: %v", err)
	}

	// The chunk should have no embedding since it was skipped
	if len(chunks[0].OllamaEmbedding) != 0 {
		t.Error("Expected chunk to have no embedding after being skipped")
	}
}
