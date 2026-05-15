# Configuration File Reference

The builder reads a single YAML configuration file that describes the
output database, the documentation sources, and the embedding
providers. The example file at
[`examples/pgedge-ai-kb-builder.yaml`](https://github.com/pgEdge/pgedge-ai-kb/blob/main/examples/pgedge-ai-kb-builder.yaml)
provides an annotated reference; the sections below describe each
field.

## Command-Line Flags

The builder supports the following flags:

| Flag | Description |
|------|-------------|
| `-c`, `--config` | Path to the configuration file. Defaults to `pgedge-ai-kb-builder.yaml` next to the binary. |
| `-d`, `--database` | Override the configured output database path. |
| `--skip-updates` | Skip the `git pull` step for previously cloned sources. |
| `--add-missing-embeddings` | Generate embeddings only for chunks missing them, instead of rebuilding. |
| `--clear-embeddings <provider>` | Clear embeddings for `openai`, `voyage`, or `ollama`. |
| `--max-retries <n>` | Retry budget for transient embedding API errors. `0` retries indefinitely (default `5`). |
| `-h`, `--help` | Print usage information. |

## Top-Level Fields

The configuration file accepts the following top-level keys:

- The `database_path` key specifies the path to the output SQLite
  database. The default is `pgedge-ai-kb.db` next to the
  configuration file.

- The `doc_source_path` key specifies the directory where the
  builder stores cloned Git repositories and other working state.
  The default is `doc-source` next to the configuration file.

- The `sources` key lists one or more documentation sources to
  process; the [Configuring Sources](../guide/sources.md) guide
  describes the available fields.

- The `embeddings` key configures the OpenAI, Voyage, and Ollama
  providers; the [Configuring Embedding
  Providers](../guide/embeddings.md) guide describes the fields per
  provider.

## Environment Variables

The builder reads API keys from files defined in the configuration. It
also recognises the following environment variables, which override the
file contents:

- The `OPENAI_API_KEY` variable overrides
  `embeddings.openai.api_key_file`.

- The `VOYAGE_API_KEY` variable overrides
  `embeddings.voyage.api_key_file`.

## Annotated Configuration Example

The following example demonstrates a complete builder configuration
file with comments for every option:

```yaml
# ============================================================================
# OUTPUT DATABASE CONFIGURATION
# ============================================================================
# Path to the output SQLite knowledgebase database.
# Default: pgedge-ai-kb.db in the same directory as this config file.
# Command-line flag: --database or -d
database_path: "pgedge-ai-kb.db"

# ============================================================================
# DOCUMENTATION SOURCE DIRECTORY
# ============================================================================
# Directory for storing downloaded/processed documentation.
# Git repositories are cloned here.
# Default: doc-source in the same directory as this config file.
doc_source_path: "doc-source"

# ============================================================================
# DOCUMENTATION SOURCES
# ============================================================================
# List of documentation sources to process. Each source is either a Git
# repository or a local path. Each entry must include project_name.
sources:
    # Example: PostgreSQL 17 documentation from Git, by branch.
    - git_url: "https://github.com/postgres/postgres.git"
      branch: "REL_17_STABLE"                # Git branch to use
      # tag: "REL_17_0"                       # Alternative: use a tag instead
      doc_path: "doc/src/sgml"                # Path within repo
      project_name: "PostgreSQL"              # Required
      project_version: "17"                   # Optional

    # Example: pgEdge documentation
    # - git_url: "https://github.com/pgEdge/docs.git"
    #   branch: "main"
    #   doc_path: "."
    #   project_name: "pgEdge"
    #   project_version: "latest"

    # Example: local documentation directory.
    # - local_path: "~/projects/my-project"
    #   doc_path: "docs"
    #   project_name: "My Project"
    #   project_version: "1.0"

# ============================================================================
# EMBEDDING PROVIDER CONFIGURATION
# ============================================================================
# Enable at least one provider. The builder calls every enabled provider
# for every chunk and stores the resulting vectors in separate columns.
embeddings:
    # OpenAI ----------------------------------------------------------------
    openai:
        enabled: true

        # Path to a file containing the OpenAI API key.
        # Environment variable OPENAI_API_KEY takes priority over this file.
        # Default: ~/.openai-api-key
        api_key_file: "~/.openai-api-key"

        # Embedding model.
        # Options: text-embedding-3-small (1536), text-embedding-3-large
        # (3072), text-embedding-ada-002 (1536).
        # Default: text-embedding-3-small
        model: "text-embedding-3-small"

        # Embedding dimensionality. Required only for models that support
        # variable dimensions. Default: 1536.
        dimensions: 1536

    # Voyage AI -------------------------------------------------------------
    voyage:
        enabled: false
        # Default: ~/.voyage-api-key
        api_key_file: "~/.voyage-api-key"
        # Options: voyage-3 (1024 dim), voyage-3-lite (512 dim).
        # Default: voyage-3
        model: "voyage-3"

    # Ollama (local) --------------------------------------------------------
    ollama:
        enabled: false
        # Ollama HTTP endpoint. Default: http://localhost:11434
        endpoint: "http://localhost:11434"
        # Options: nomic-embed-text (768 dim), mxbai-embed-large (1024 dim).
        # Note: the model must be pulled with `ollama pull <model>` first.
        # Default: nomic-embed-text
        model: "nomic-embed-text"
        # Optional, only required for Ollama Cloud.
        # api_key_file: "~/.ollama-api-key"
```

## Supported Document Formats

The builder recognises files by extension and converts each to Markdown
before chunking. The following formats are supported:

- Markdown (`.md`) passes through with title extraction.

- HTML (`.html`, `.htm`) converts via the `html-to-markdown` library.

- reStructuredText (`.rst`) converts through a built-in
  pattern-based parser.

- SGML (`.sgml`, `.sgm`) converts through a DocBook-aware parser.

- DocBook XML (`.xml`) converts through the same parser as SGML.

The chunker preserves structural elements (code blocks, tables, lists,
blockquotes) when their size permits.

## See Also

- [Configuring Sources](../guide/sources.md) covers source fields in
  detail.

- [Configuring Embedding Providers](../guide/embeddings.md) covers
  every provider option.

- [Architecture](../contributing/architecture.md) explains how the
  builder processes the configuration end-to-end.
