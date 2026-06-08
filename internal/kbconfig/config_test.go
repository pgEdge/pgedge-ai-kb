/*-------------------------------------------------------------------------
 *
 * pgEdge AI Knowledgebase Builder
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 *-------------------------------------------------------------------------
 */

package kbconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `
database_path: "test.db"
doc_source_path: "docs"

sources:
  - local_path: "/tmp/test-docs"
    project_name: "Test Project"
    project_version: "1.0"

embeddings:
  openai:
    enabled: true
    api_key_file: "/tmp/fake-key"
    model: "text-embedding-3-small"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Create fake API key file
	keyPath := "/tmp/fake-key"
	if err := os.WriteFile(keyPath, []byte("test-api-key"), 0644); err != nil {
		t.Fatalf("Failed to write test key: %v", err)
	}
	defer os.Remove(keyPath)

	// Load config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded values
	if cfg.DatabasePath == "" {
		t.Error("DatabasePath should be set")
	}

	if cfg.DocSourcePath == "" {
		t.Error("DocSourcePath should be set")
	}

	if len(cfg.Sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(cfg.Sources))
	}

	if cfg.Sources[0].ProjectName != "Test Project" {
		t.Errorf("Expected project name 'Test Project', got '%s'", cfg.Sources[0].ProjectName)
	}

	if !cfg.Embeddings.OpenAI.Enabled {
		t.Error("OpenAI embeddings should be enabled")
	}

	if cfg.Embeddings.OpenAI.APIKey != "test-api-key" {
		t.Errorf("API key should be loaded, got '%s'", cfg.Embeddings.OpenAI.APIKey)
	}

	// Timeouts are unset in the config above, so the defaults apply and
	// parse into durations.
	if cfg.Embeddings.RequestTimeoutDuration != 10*time.Minute {
		t.Errorf("Expected default request timeout 10m, got %s",
			cfg.Embeddings.RequestTimeoutDuration)
	}
	if cfg.Embeddings.PerAttemptTimeoutDuration != 90*time.Second {
		t.Errorf("Expected default per-attempt timeout 90s, got %s",
			cfg.Embeddings.PerAttemptTimeoutDuration)
	}
}

// TestLoadConfigTimeouts covers parsing of explicit timeout values and
// rejection of an unparseable duration string.
func TestLoadConfigTimeouts(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "fake-key")
	if err := os.WriteFile(keyPath, []byte("test-api-key"), 0600); err != nil {
		t.Fatalf("Failed to write test key: %v", err)
	}

	header := `
sources:
  - local_path: "/tmp/test-docs"
    project_name: "Test Project"
embeddings:
  openai:
    enabled: true
    api_key_file: "` + keyPath + `"
`

	t.Run("explicit values are parsed", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "explicit.yaml")
		content := header + "  request_timeout: \"5m\"\n  per_attempt_timeout: \"30s\"\n"
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}
		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load returned error: %v", err)
		}
		if cfg.Embeddings.RequestTimeoutDuration != 5*time.Minute {
			t.Errorf("Expected 5m, got %s", cfg.Embeddings.RequestTimeoutDuration)
		}
		if cfg.Embeddings.PerAttemptTimeoutDuration != 30*time.Second {
			t.Errorf("Expected 30s, got %s", cfg.Embeddings.PerAttemptTimeoutDuration)
		}
	})

	t.Run("unparseable duration is rejected", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "bad.yaml")
		content := header + "  request_timeout: \"banana\"\n"
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}
		if _, err := Load(configPath); err == nil {
			t.Error("Expected error for unparseable request_timeout, got none")
		}
	})
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		shouldError bool
	}{
		{
			name: "valid config",
			config: &Config{
				Sources: []DocumentSource{
					{
						LocalPath:      "/tmp/test",
						ProjectName:    "Test",
						ProjectVersion: "1.0",
					},
				},
				Embeddings: EmbeddingConfig{
					OpenAI:                    OpenAIConfig{Enabled: true},
					RequestTimeoutDuration:    10 * time.Minute,
					PerAttemptTimeoutDuration: 90 * time.Second,
				},
			},
			shouldError: false,
		},
		{
			name: "valid config with only gemini",
			config: &Config{
				Sources: []DocumentSource{
					{
						LocalPath:      "/tmp/test",
						ProjectName:    "Test",
						ProjectVersion: "1.0",
					},
				},
				Embeddings: EmbeddingConfig{
					Gemini:                    GeminiConfig{Enabled: true},
					RequestTimeoutDuration:    10 * time.Minute,
					PerAttemptTimeoutDuration: 90 * time.Second,
				},
			},
			shouldError: false,
		},
		{
			name: "no sources",
			config: &Config{
				Sources: []DocumentSource{},
				Embeddings: EmbeddingConfig{
					OpenAI: OpenAIConfig{Enabled: true},
				},
			},
			shouldError: true,
		},
		{
			name: "no embedding providers",
			config: &Config{
				Sources: []DocumentSource{
					{
						LocalPath:      "/tmp/test",
						ProjectName:    "Test",
						ProjectVersion: "1.0",
					},
				},
				Embeddings: EmbeddingConfig{},
			},
			shouldError: true,
		},
		{
			name: "missing project name",
			config: &Config{
				Sources: []DocumentSource{
					{
						LocalPath:      "/tmp/test",
						ProjectVersion: "1.0",
					},
				},
				Embeddings: EmbeddingConfig{
					OpenAI: OpenAIConfig{Enabled: true},
				},
			},
			shouldError: true,
		},
		{
			name: "both git and local",
			config: &Config{
				Sources: []DocumentSource{
					{
						GitURL:         "https://github.com/test/test",
						LocalPath:      "/tmp/test",
						ProjectName:    "Test",
						ProjectVersion: "1.0",
					},
				},
				Embeddings: EmbeddingConfig{
					OpenAI: OpenAIConfig{Enabled: true},
				},
			},
			shouldError: true,
		},
		{
			name: "missing project version is allowed",
			config: &Config{
				Sources: []DocumentSource{
					{
						LocalPath:   "/tmp/test",
						ProjectName: "Test",
						// ProjectVersion is intentionally omitted
					},
				},
				Embeddings: EmbeddingConfig{
					OpenAI:                    OpenAIConfig{Enabled: true},
					RequestTimeoutDuration:    10 * time.Minute,
					PerAttemptTimeoutDuration: 90 * time.Second,
				},
			},
			shouldError: false,
		},
		{
			name: "per_attempt_timeout not below request_timeout",
			config: &Config{
				Sources: []DocumentSource{
					{
						LocalPath:      "/tmp/test",
						ProjectName:    "Test",
						ProjectVersion: "1.0",
					},
				},
				Embeddings: EmbeddingConfig{
					OpenAI:                    OpenAIConfig{Enabled: true},
					RequestTimeout:            "90s",
					PerAttemptTimeout:         "90s",
					RequestTimeoutDuration:    90 * time.Second,
					PerAttemptTimeoutDuration: 90 * time.Second,
				},
			},
			shouldError: true,
		},
		{
			name: "non-positive request_timeout",
			config: &Config{
				Sources: []DocumentSource{
					{
						LocalPath:      "/tmp/test",
						ProjectName:    "Test",
						ProjectVersion: "1.0",
					},
				},
				Embeddings: EmbeddingConfig{
					OpenAI:                    OpenAIConfig{Enabled: true},
					PerAttemptTimeoutDuration: 90 * time.Second,
				},
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.config)
			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "tilde expansion",
			input:    "~/test",
			contains: "/test",
		},
		{
			name:     "tilde only",
			input:    "~",
			contains: "",
		},
		{
			name:     "absolute path",
			input:    "/tmp/test",
			contains: "/tmp/test",
		},
		{
			name:     "relative path",
			input:    "test",
			contains: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input)
			if tt.input == "~" {
				// Should expand to home directory
				if result == tt.input {
					t.Error("Tilde should be expanded")
				}
			} else if tt.contains != "" && result != tt.input {
				// Check that expanded path contains the expected part
				if !strings.Contains(result, tt.contains) {
					t.Errorf("Expected path to contain '%s', got: %s", tt.contains, result)
				}
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	cfg := &Config{
		Embeddings: EmbeddingConfig{
			OpenAI: OpenAIConfig{Enabled: true},
			Voyage: VoyageConfig{Enabled: true},
			Ollama: OllamaConfig{Enabled: true},
			Gemini: GeminiConfig{Enabled: true},
		},
	}

	err := applyDefaults(cfg, configPath)
	if err != nil {
		t.Fatalf("applyDefaults failed: %v", err)
	}

	// Check defaults were applied
	if cfg.DatabasePath == "" {
		t.Error("DatabasePath should have default")
	}

	if cfg.DocSourcePath == "" {
		t.Error("DocSourcePath should have default")
	}

	if cfg.Embeddings.OpenAI.Model == "" {
		t.Error("OpenAI model should have default")
	}

	if cfg.Embeddings.OpenAI.Dimensions == 0 {
		t.Error("OpenAI dimensions should have default")
	}

	if cfg.Embeddings.Voyage.Model == "" {
		t.Error("Voyage model should have default")
	}

	if cfg.Embeddings.Ollama.Model == "" {
		t.Error("Ollama model should have default")
	}

	if cfg.Embeddings.Ollama.Endpoint == "" {
		t.Error("Ollama endpoint should have default")
	}

	if cfg.Embeddings.Gemini.Model != "gemini-embedding-001" {
		t.Errorf("Gemini.Model = %q, want gemini-embedding-001",
			cfg.Embeddings.Gemini.Model)
	}

	if cfg.Embeddings.Gemini.APIKeyFile == "" {
		t.Error("Gemini.APIKeyFile should have default")
	}
}
