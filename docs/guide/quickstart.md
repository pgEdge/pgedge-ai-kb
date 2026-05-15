# Quick Start

The Quick Start guide walks you through installing the builder, writing
a minimal configuration, and producing your first knowledgebase
database. The end-to-end run completes in a few minutes.

## Prerequisites

The builder requires the following tools on your build machine:

- Go 1.25 or later, with the `go` binary on `$PATH`.

- Git, used for cloning documentation source repositories.

- One embedding provider, with credentials or a local Ollama
  instance. The [Configuring Embedding
  Providers](embeddings.md) guide describes the options.

## Build the Binary

Clone the repository and build the binary with `make build`. The build
produces a pure-Go binary that has no runtime system dependencies:

```bash
git clone https://github.com/pgEdge/pgedge-ai-kb.git
cd pgedge-ai-kb
make build
```

The build writes `bin/pgedge-ai-kb-builder`. You can also install it to
your `GOPATH` with `make install`.

## Provide an API Key

For the simplest setup, generate a knowledgebase using OpenAI
embeddings. Store the API key in a file with restrictive permissions
so the builder can read it later:

```bash
echo "sk-your-openai-key" > ~/.openai-api-key
chmod 600 ~/.openai-api-key
```

Voyage AI and Ollama follow the same file-based key pattern; see
[Configuring Embedding Providers](embeddings.md) for details.

## Write a Minimal Configuration

Copy the bundled example and edit it to match your sources. The
following minimal configuration ingests the PostgreSQL 17
documentation and produces an OpenAI-only knowledgebase:

```yaml
database_path: "bin/pgedge-ai-kb.db"
doc_source_path: "bin/doc-source"

sources:
    - git_url: "https://github.com/postgres/postgres.git"
      branch: "REL_17_STABLE"
      doc_path: "doc/src/sgml"
      project_name: "PostgreSQL"
      project_version: "17"

embeddings:
    openai:
        enabled: true
        api_key_file: "~/.openai-api-key"
        model: "text-embedding-3-small"
        dimensions: 1536

    voyage:
        enabled: false

    ollama:
        enabled: false
```

Save the file as `quickstart.yaml` in the repository root.

## Run the Build

Run the builder against your configuration:

```bash
./bin/pgedge-ai-kb-builder --config quickstart.yaml
```

The builder clones the PostgreSQL repository, converts the SGML
documentation to Markdown, chunks the content, calls the OpenAI
embeddings API, and writes `bin/pgedge-ai-kb.db`. The run logs
progress per file and prints summary statistics on completion.

## Verify the Output

Confirm the database exists and inspect a few stats with the
`sqlite3` CLI:

```bash
ls -lh bin/pgedge-ai-kb.db
sqlite3 bin/pgedge-ai-kb.db "SELECT COUNT(*) FROM chunks;"
sqlite3 bin/pgedge-ai-kb.db \
    "SELECT project_name, project_version, COUNT(*)
     FROM chunks GROUP BY 1, 2;"
```

The first query returns the total chunk count for the build. The
second query lists chunks per project and version.

## Next Steps

The following pages describe richer setups:

- [Building a Knowledgebase](building.md) walks through writing custom
  domain documentation and combining it with upstream sources.

- [Configuring Sources](sources.md) explains every source option,
  including branch and tag selection.

- [Configuring Embedding Providers](embeddings.md) describes how to
  enable multiple providers in a single build.

- [Configuration File Reference](../reference/config.md) lists every
  configuration field with comments.
