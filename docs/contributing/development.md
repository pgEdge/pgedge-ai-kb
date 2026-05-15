# Development Setup

This page describes how to set up a development environment for the
pgEdge AI Knowledgebase Builder.

## Prerequisites

The builder uses pure Go and has no native dependencies. The
development environment requires the following tools:

- Go 1.25 or later, with `$GOPATH/bin` on your `$PATH`.

- Git.

- GNU make.

- golangci-lint v2.x for static analysis.

- Python 3.11 or later for building the documentation site.

Install golangci-lint with:

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
```

## Cloning the Repository

Clone the repository and confirm the toolchain works:

```bash
git clone https://github.com/pgEdge/pgedge-ai-kb.git
cd pgedge-ai-kb
go version
go mod download
```

## Building Locally

The default Makefile target builds the binary:

```bash
make build
```

The build produces `bin/pgedge-ai-kb-builder`. Cross-compile for all
platforms with `make build-all`.

## Running Tests

The `test` target runs every unit test:

```bash
make test
```

The `test-coverage` target prints per-function coverage:

```bash
make test-coverage
```

## Running the Linter

The `lint` target runs golangci-lint with the project configuration
(`.golangci.yml`):

```bash
make lint
```

Set `make lint` clean before opening a pull request.

## Formatting Code

The Makefile exposes `fmt` and `gofmt` targets:

```bash
make fmt
make gofmt
```

Both targets format every Go file under `cmd/` and `internal/`. CI
fails when `gofmt -l` lists any file.

## Working with Documentation

The documentation site uses MkDocs. The `docs` target creates a
virtualenv on first run and builds the site:

```bash
make docs
```

The built site lands under `site/`. Iterate locally with:

```bash
source venv/bin/activate
mkdocs serve
```

`mkdocs serve` watches the source tree and reloads the browser on
every change.

## Building a Knowledgebase Locally

The `kb` target builds `bin/pgedge-ai-kb.db` from the bundled example
configuration:

```bash
make kb
```

This requires functional embedding provider credentials (see
[Configuring Embedding Providers](../guide/embeddings.md)).

## Project Layout

The repository is organised as follows:

- The `cmd/kb-builder/` directory holds the CLI entry point.

- The `internal/kbchunker/` directory holds the document chunking
  pipeline.

- The `internal/kbconfig/` directory holds configuration parsing and
  validation.

- The `internal/kbconverter/` directory holds the per-format Markdown
  converters.

- The `internal/kbdatabase/` directory holds the SQLite database
  layer.

- The `internal/kbembed/` directory holds the embedding providers.

- The `internal/kbsource/` directory holds the Git and local source
  fetchers.

- The `internal/kbtypes/` directory holds shared type definitions.

- The `docs/` directory holds the MkDocs source.

- The `examples/` directory holds the canonical example configuration.

- The `.github/workflows/` directory holds CI and release workflows.

The [Architecture](architecture.md) document describes the
relationships between these packages.

## See Also

- [Testing](testing.md) explains how the test suite is organised.

- [CI/CD](ci-cd.md) documents the automation pipelines.

- [Architecture](architecture.md) covers the internal design.
