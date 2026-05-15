# pgEdge AI Knowledgebase Builder

[![CI - KB Builder](https://github.com/pgEdge/pgedge-ai-kb/actions/workflows/ci-kb-builder.yml/badge.svg?branch=main)](https://github.com/pgEdge/pgedge-ai-kb/actions/workflows/ci-kb-builder.yml?query=branch%3Amain)
[![CI - Documentation](https://github.com/pgEdge/pgedge-ai-kb/actions/workflows/ci-docs.yml/badge.svg?branch=main)](https://github.com/pgEdge/pgedge-ai-kb/actions/workflows/ci-docs.yml?query=branch%3Amain)

- About the Knowledgebase Builder
    - [pgEdge AI Knowledgebase Builder](docs/index.md)
- User Guide
    - [Quick Start](docs/guide/quickstart.md)
    - [Installation](docs/guide/installation.md)
    - [Building a Knowledgebase](docs/guide/building.md)
    - [Configuring Sources](docs/guide/sources.md)
    - [Configuring Embedding Providers](docs/guide/embeddings.md)
    - [Output Database Layout](docs/guide/output-db.md)
    - [Troubleshooting](docs/guide/troubleshooting.md)
- Reference
    - [Configuration File Reference](docs/reference/config.md)
- Contributing
    - [Development Setup](docs/contributing/development.md)
    - [Architecture](docs/contributing/architecture.md)
    - [Testing](docs/contributing/testing.md)
    - [CI/CD](docs/contributing/ci-cd.md)
- [Release Notes](docs/changelog.md)
- [Licence](docs/LICENSE.md)

The pgEdge AI Knowledgebase Builder processes documentation from Git
repositories and local paths, then assembles a searchable SQLite database
with vector embeddings. The resulting database powers retrieval-augmented
features in pgEdge tools such as the pgEdge Postgres MCP Server and the
pgEdge AI DBA Workbench.

The builder converts content from multiple formats (Markdown, HTML,
reStructuredText, SGML, DocBook XML), chunks documents intelligently,
generates embeddings using OpenAI, Voyage AI, or Ollama, and stores
everything in an optimized SQLite database.

## Quick Start

The [Quick Start](docs/guide/quickstart.md) guide covers installation
and first build. In short:

```bash
git clone https://github.com/pgEdge/pgedge-ai-kb.git
cd pgedge-ai-kb
make build
cp examples/pgedge-ai-kb-builder.yaml ./pgedge-ai-kb-builder.yaml
# Edit pgedge-ai-kb-builder.yaml; provide API keys as documented
./bin/pgedge-ai-kb-builder --config pgedge-ai-kb-builder.yaml
```

The build produces `pgedge-ai-kb.db`, a SQLite knowledgebase database that
downstream consumers load for semantic search.

## Key Features

- **Multi-source ingestion** - Pull documentation from Git repositories
  (by branch or tag) or local paths.
- **Multi-format conversion** - Parse Markdown, HTML, reStructuredText,
  SGML, and DocBook XML, then normalise to Markdown.
- **Intelligent chunking** - Heading-aware splitting that preserves
  document structure for retrieval quality.
- **Multiple embedding providers** - Generate embeddings with OpenAI,
  Voyage AI, or Ollama (local); enable any subset.
- **Incremental builds** - SHA256 checksums skip unchanged files and
  reuse existing chunks across versions.
- **Resilient API calls** - Configurable retries with exponential
  backoff for transient embedding API errors.
- **Single binary** - Pure-Go build produces a portable CLI binary.

## Development

### Prerequisites

- Go 1.25 or higher
- golangci-lint v2.x (for linting)
- Python 3.11+ (for documentation builds)

### Setup Linter

The project uses golangci-lint v2.x. Install it with:

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
```

### Building

```bash
git clone https://github.com/pgEdge/pgedge-ai-kb.git
cd pgedge-ai-kb
make build
```

### Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run linting
make lint
```

### Documentation

Build the documentation site locally:

```bash
make docs
# Open site/index.html
```

## Support

To report an issue with the software, visit:
[GitHub Issues](https://github.com/pgEdge/pgedge-ai-kb/issues)

For more information, visit
[docs.pgedge.com](https://docs.pgedge.com)

This project is licensed under the
[PostgreSQL License](LICENSE.md).
