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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Default embedding request timeouts. RequestTimeout is the overall
// ceiling for one embedding request including all retries;
// PerAttemptTimeout bounds each individual HTTP attempt so a stalled
// attempt is retried rather than fatally cancelling the request. The
// per-attempt value sits below the overall ceiling to leave room for
// retries; Gemini's heavy batchEmbedContents motivated these defaults.
const (
	defaultRequestTimeout    = "10m"
	defaultPerAttemptTimeout = "90s"
)

// Config represents the kb-builder configuration
type Config struct {
	// Output database path
	DatabasePath string `yaml:"database_path"`

	// Directory for storing downloaded/processed documentation
	DocSourcePath string `yaml:"doc_source_path"`

	// Documentation sources
	Sources []DocumentSource `yaml:"sources"`

	// Embedding provider configurations
	Embeddings EmbeddingConfig `yaml:"embeddings"`
}

// DocumentSource represents a source of documentation
type DocumentSource struct {
	// For Git repositories
	GitURL string `yaml:"git_url,omitempty"`
	Branch string `yaml:"branch,omitempty"`
	Tag    string `yaml:"tag,omitempty"`

	// For local paths
	LocalPath string `yaml:"local_path,omitempty"`

	// Common fields
	DocPath        string `yaml:"doc_path"`        // Path within project containing docs
	ProjectName    string `yaml:"project_name"`    // User-defined project name
	ProjectVersion string `yaml:"project_version"` // User-defined version
}

// EmbeddingConfig contains configuration for all embedding providers
type EmbeddingConfig struct {
	OpenAI OpenAIConfig `yaml:"openai"`
	Voyage VoyageConfig `yaml:"voyage"`
	Ollama OllamaConfig `yaml:"ollama"`
	Gemini GeminiConfig `yaml:"gemini"`

	// RequestTimeout is the overall wall-clock ceiling for one embedding
	// request, including all retries (e.g. "10m"). PerAttemptTimeout
	// bounds each individual HTTP attempt (e.g. "90s"); a stalled attempt
	// is retried rather than cancelling the whole request. Set
	// PerAttemptTimeout below RequestTimeout to leave room for retries.
	RequestTimeout    string `yaml:"request_timeout,omitempty"`
	PerAttemptTimeout string `yaml:"per_attempt_timeout,omitempty"`

	// Parsed forms of the timeouts above. Populated at load time, not
	// from YAML.
	RequestTimeoutDuration    time.Duration `yaml:"-"`
	PerAttemptTimeoutDuration time.Duration `yaml:"-"`
}

// OpenAIConfig contains OpenAI embedding configuration
type OpenAIConfig struct {
	Enabled    bool   `yaml:"enabled"`
	APIKeyFile string `yaml:"api_key_file"`
	APIKey     string // Loaded at runtime, not from YAML
	Model      string `yaml:"model"`      // e.g., "text-embedding-3-small"
	Dimensions int    `yaml:"dimensions"` // Optional, model-specific
}

// VoyageConfig contains Voyage AI embedding configuration
type VoyageConfig struct {
	Enabled    bool   `yaml:"enabled"`
	APIKeyFile string `yaml:"api_key_file"`
	APIKey     string // Loaded at runtime, not from YAML
	Model      string `yaml:"model"` // e.g., "voyage-3"
}

// OllamaConfig contains Ollama embedding configuration
type OllamaConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Endpoint      string `yaml:"endpoint"`       // e.g., "http://localhost:11434"
	Model         string `yaml:"model"`          // e.g., "nomic-embed-text"
	ContextLength int    `yaml:"context_length"` // Context window size (num_ctx)
	APIKeyFile    string `yaml:"api_key_file"`   // Optional, only needed for Ollama Cloud
	APIKey        string // Loaded at runtime, not from YAML
}

// GeminiConfig contains Gemini embedding configuration
type GeminiConfig struct {
	Enabled    bool   `yaml:"enabled"`
	APIKeyFile string `yaml:"api_key_file"`
	APIKey     string // Loaded at runtime, not from YAML
	Model      string `yaml:"model"` // e.g., "gemini-embedding-001"
}

// Load reads and parses the configuration file
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	if err := applyDefaults(&config, configPath); err != nil {
		return nil, err
	}

	// Validate configuration
	if err := validate(&config); err != nil {
		return nil, err
	}

	// Load API keys
	if err := loadAPIKeys(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// applyDefaults sets default values for unspecified config fields
func applyDefaults(config *Config, configPath string) error {
	configDir := filepath.Dir(configPath)

	// Default database path
	if config.DatabasePath == "" {
		config.DatabasePath = filepath.Join(configDir, "pgedge-ai-kb.db")
	}

	// Default doc source path
	if config.DocSourcePath == "" {
		config.DocSourcePath = filepath.Join(configDir, "doc-source")
	}

	// Default OpenAI settings
	if config.Embeddings.OpenAI.Enabled {
		if config.Embeddings.OpenAI.APIKeyFile == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			config.Embeddings.OpenAI.APIKeyFile = filepath.Join(home, ".openai-api-key")
		}
		if config.Embeddings.OpenAI.Model == "" {
			config.Embeddings.OpenAI.Model = "text-embedding-3-small"
		}
		if config.Embeddings.OpenAI.Dimensions == 0 {
			config.Embeddings.OpenAI.Dimensions = 1536
		}
	}

	// Default Voyage settings
	if config.Embeddings.Voyage.Enabled {
		if config.Embeddings.Voyage.APIKeyFile == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			config.Embeddings.Voyage.APIKeyFile = filepath.Join(home, ".voyage-api-key")
		}
		if config.Embeddings.Voyage.Model == "" {
			config.Embeddings.Voyage.Model = "voyage-3"
		}
	}

	// Default Ollama settings
	if config.Embeddings.Ollama.Enabled {
		if config.Embeddings.Ollama.Endpoint == "" {
			config.Embeddings.Ollama.Endpoint = "http://localhost:11434"
		}
		if config.Embeddings.Ollama.Model == "" {
			config.Embeddings.Ollama.Model = "nomic-embed-text"
		}
		if config.Embeddings.Ollama.ContextLength == 0 {
			// Default to 8192 tokens - nomic-embed-text v1.5 supports up to 8192
			// This provides headroom since our chunks target ~250 words which
			// can translate to 750+ tokens with subword tokenization (technical
			// content with long terms can tokenize to 3-4x more than word count)
			config.Embeddings.Ollama.ContextLength = 8192
		}
	}

	// Default Gemini settings
	if config.Embeddings.Gemini.Enabled {
		if config.Embeddings.Gemini.APIKeyFile == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			config.Embeddings.Gemini.APIKeyFile = filepath.Join(home, ".gemini-api-key")
		}
		if config.Embeddings.Gemini.Model == "" {
			config.Embeddings.Gemini.Model = "gemini-embedding-001"
		}
	}

	// Default embedding request timeouts and parse them into durations.
	if config.Embeddings.RequestTimeout == "" {
		config.Embeddings.RequestTimeout = defaultRequestTimeout
	}
	if config.Embeddings.PerAttemptTimeout == "" {
		config.Embeddings.PerAttemptTimeout = defaultPerAttemptTimeout
	}
	reqTimeout, err := time.ParseDuration(config.Embeddings.RequestTimeout)
	if err != nil {
		return fmt.Errorf("invalid request_timeout %q: %w",
			config.Embeddings.RequestTimeout, err)
	}
	attemptTimeout, err := time.ParseDuration(config.Embeddings.PerAttemptTimeout)
	if err != nil {
		return fmt.Errorf("invalid per_attempt_timeout %q: %w",
			config.Embeddings.PerAttemptTimeout, err)
	}
	config.Embeddings.RequestTimeoutDuration = reqTimeout
	config.Embeddings.PerAttemptTimeoutDuration = attemptTimeout

	// Expand paths with ~
	config.DatabasePath = expandPath(config.DatabasePath)
	config.DocSourcePath = expandPath(config.DocSourcePath)
	if config.Embeddings.OpenAI.APIKeyFile != "" {
		config.Embeddings.OpenAI.APIKeyFile = expandPath(config.Embeddings.OpenAI.APIKeyFile)
	}
	if config.Embeddings.Voyage.APIKeyFile != "" {
		config.Embeddings.Voyage.APIKeyFile = expandPath(config.Embeddings.Voyage.APIKeyFile)
	}
	if config.Embeddings.Ollama.APIKeyFile != "" {
		config.Embeddings.Ollama.APIKeyFile = expandPath(config.Embeddings.Ollama.APIKeyFile)
	}
	if config.Embeddings.Gemini.APIKeyFile != "" {
		config.Embeddings.Gemini.APIKeyFile = expandPath(config.Embeddings.Gemini.APIKeyFile)
	}

	return nil
}

// validate checks that the configuration is valid
func validate(config *Config) error {
	if len(config.Sources) == 0 {
		return fmt.Errorf("no documentation sources configured")
	}

	for i, source := range config.Sources {
		// Check that either Git or local path is specified
		hasGit := source.GitURL != ""
		hasLocal := source.LocalPath != ""

		if !hasGit && !hasLocal {
			return fmt.Errorf("source %d: must specify either git_url or local_path", i)
		}
		if hasGit && hasLocal {
			return fmt.Errorf("source %d: cannot specify both git_url and local_path", i)
		}

		// Check required fields
		if source.ProjectName == "" {
			return fmt.Errorf("source %d: project_name is required", i)
		}
		// Note: project_version is optional (some docs have no specific version)
	}

	// Check that at least one embedding provider is enabled
	if !config.Embeddings.OpenAI.Enabled &&
		!config.Embeddings.Voyage.Enabled &&
		!config.Embeddings.Ollama.Enabled &&
		!config.Embeddings.Gemini.Enabled {
		return fmt.Errorf("at least one embedding provider must be enabled")
	}

	// Both timeouts must be positive, and the per-attempt ceiling must
	// sit below the overall ceiling so retries have room to run.
	if config.Embeddings.RequestTimeoutDuration <= 0 {
		return fmt.Errorf("request_timeout must be positive, got %q",
			config.Embeddings.RequestTimeout)
	}
	if config.Embeddings.PerAttemptTimeoutDuration <= 0 {
		return fmt.Errorf("per_attempt_timeout must be positive, got %q",
			config.Embeddings.PerAttemptTimeout)
	}
	if config.Embeddings.PerAttemptTimeoutDuration >=
		config.Embeddings.RequestTimeoutDuration {
		return fmt.Errorf(
			"per_attempt_timeout (%s) must be less than request_timeout (%s)",
			config.Embeddings.PerAttemptTimeout,
			config.Embeddings.RequestTimeout)
	}

	return nil
}

// loadAPIKeys reads API keys from files
func loadAPIKeys(config *Config) error {
	if config.Embeddings.OpenAI.Enabled {
		key, err := readAPIKey(config.Embeddings.OpenAI.APIKeyFile)
		if err != nil {
			return fmt.Errorf("OpenAI API key: %w", err)
		}
		config.Embeddings.OpenAI.APIKey = key
	}

	if config.Embeddings.Voyage.Enabled {
		key, err := readAPIKey(config.Embeddings.Voyage.APIKeyFile)
		if err != nil {
			return fmt.Errorf("voyage API key: %w", err)
		}
		config.Embeddings.Voyage.APIKey = key
	}

	// Ollama API key is optional: only needed for Ollama Cloud, not
	// for local Ollama deployments.
	if config.Embeddings.Ollama.Enabled && config.Embeddings.Ollama.APIKeyFile != "" {
		key, err := readAPIKey(config.Embeddings.Ollama.APIKeyFile)
		if err != nil {
			return fmt.Errorf("ollama API key: %w", err)
		}
		config.Embeddings.Ollama.APIKey = key
	}

	if config.Embeddings.Gemini.Enabled {
		key, err := readAPIKey(config.Embeddings.Gemini.APIKeyFile)
		if err != nil {
			return fmt.Errorf("gemini API key: %w", err)
		}
		config.Embeddings.Gemini.APIKey = key
	}

	return nil
}

// readAPIKey reads an API key from a file
func readAPIKey(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read API key file %s: %w", path, err)
	}

	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", fmt.Errorf("API key file %s is empty", path)
	}

	return key, nil
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}

	return path
}

// Target identifies one provider/model combination that produces its
// own output database. Provider is the canonical lowercase key used by
// the embedding generator and database columns; Label is the
// human-facing name; Model is the configured model name.
type Target struct {
	Provider string
	Label    string
	Model    string
}

// EnabledTargets returns the enabled provider/model targets in a stable
// order (openai, voyage, ollama, gemini). Models are populated by
// applyDefaults for every enabled provider.
func (c *Config) EnabledTargets() []Target {
	var targets []Target
	if c.Embeddings.OpenAI.Enabled {
		targets = append(targets, Target{"openai", "OpenAI", c.Embeddings.OpenAI.Model})
	}
	if c.Embeddings.Voyage.Enabled {
		targets = append(targets, Target{"voyage", "Voyage", c.Embeddings.Voyage.Model})
	}
	if c.Embeddings.Ollama.Enabled {
		targets = append(targets, Target{"ollama", "Ollama", c.Embeddings.Ollama.Model})
	}
	if c.Embeddings.Gemini.Enabled {
		targets = append(targets, Target{"gemini", "Gemini", c.Embeddings.Gemini.Model})
	}
	return targets
}

// TargetForProvider returns the enabled target for the given provider
// key, or ok=false if that provider is not enabled.
func (c *Config) TargetForProvider(provider string) (Target, bool) {
	for _, t := range c.EnabledTargets() {
		if t.Provider == provider {
			return t, true
		}
	}
	return Target{}, false
}

// DatabasePathFor derives a target's output database path from the
// configured DatabasePath template:
//
//	<dir>/<stem>-<provider>-<sanitizedModel>.db
//
// where <stem> is the DatabasePath basename without its extension. For a
// DatabasePath of "bin/kb.db" and the OpenAI target this yields
// "bin/kb-openai-text-embedding-3-small.db".
func (c *Config) DatabasePathFor(t Target) string {
	dir := filepath.Dir(c.DatabasePath)
	base := filepath.Base(c.DatabasePath)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	name := fmt.Sprintf("%s-%s-%s.db", stem, t.Provider, sanitizeModel(t.Model))
	return filepath.Join(dir, name)
}

// sanitizeModel replaces every character outside [A-Za-z0-9._-] with '-'
// so a model name is safe to embed in a filename. Each unsafe character
// maps to a single '-'.
func sanitizeModel(model string) string {
	var b strings.Builder
	for _, r := range model {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}

// ForProvider returns a shallow copy of the config with only the named
// provider enabled. Embeddings is a value field, so toggling the copy's
// Enabled flags does not affect the receiver. Intended for callers that
// already know the provider is enabled (e.g. iterating EnabledTargets).
func (c *Config) ForProvider(provider string) *Config {
	clone := *c
	clone.Embeddings.OpenAI.Enabled = false
	clone.Embeddings.Voyage.Enabled = false
	clone.Embeddings.Ollama.Enabled = false
	clone.Embeddings.Gemini.Enabled = false
	switch provider {
	case "openai":
		clone.Embeddings.OpenAI.Enabled = true
	case "voyage":
		clone.Embeddings.Voyage.Enabled = true
	case "ollama":
		clone.Embeddings.Ollama.Enabled = true
	case "gemini":
		clone.Embeddings.Gemini.Enabled = true
	}
	return &clone
}
