# Architecture

This page describes the internal architecture of the pgEdge AI
Knowledgebase Builder. The intended audience is project contributors
and integrators.

## Overview

The builder is a single Go binary that performs the following work:

1. The builder fetches documentation from configured sources (Git
   repositories or local paths).

2. The builder converts every supported document format to Markdown.

3. The builder chunks each document with structural-element
   preservation and heading hierarchy tracking.

4. The builder repeats the chunking and embedding work once per
   enabled provider, generating that provider's embeddings.

5. The builder stores each provider's chunks and embeddings in its own
   optimised SQLite database, named `<stem>-<provider>-<model>.db`.

## Build Pipeline

```text
┌────────────────────────────────────────────────────────────┐
│                  pgedge-ai-kb-builder                      │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  ┌──────────────┐      ┌──────────────┐                    │
│  │  CLI Parser  │─────▶│ Config Loader│                    │
│  └──────────────┘      └──────┬───────┘                    │
│                               │                            │
│  ┌─────────────────────────────▼──────────────────────┐    │
│  │            Source Fetcher (kbsource)               │    │
│  │  • Git clone/pull with branch/tag support          │    │
│  │  • Local directory scanning                        │    │
│  └─────────────────────────┬──────────────────────────┘    │
│                            │                               │
│  ┌─────────────────────────▼──────────────────────────┐    │
│  │         Document Converter (kbconverter)           │    │
│  │  • HTML → Markdown                                 │    │
│  │  • RST → Markdown                                  │    │
│  │  • SGML/DocBook → Markdown                         │    │
│  │  • Markdown (passthrough with title extraction)    │    │
│  └─────────────────────────┬──────────────────────────┘    │
│                            │                               │
│  ┌─────────────────────────▼──────────────────────────┐    │
│  │           Document Chunker (kbchunker)             │    │
│  │  • Hybrid two-pass chunking algorithm              │    │
│  │  • 250-word target, 300 max, 3000 chars max        │    │
│  │  • Structural element preservation                 │    │
│  │  • Full heading hierarchy tracking                 │    │
│  └─────────────────────────┬──────────────────────────┘    │
│                            │                               │
│  ┌─────────────────────────▼──────────────────────────┐    │
│  │        Embedding Generator (kbembed)               │    │
│  │  • OpenAI API (batch processing)                   │    │
│  │  • Voyage AI API (batch processing)                │    │
│  │  • Ollama (sequential processing)                  │    │
│  │  • Gemini API (batch processing)                   │    │
│  └─────────────────────────┬──────────────────────────┘    │
│                            │                               │
│  ┌─────────────────────────▼──────────────────────────┐    │
│  │          Database Writer (kbdatabase)              │    │
│  │  • SQLite with transaction batching                │    │
│  │  • BLOB storage for embeddings                     │    │
│  │  • Indexes for project/version filtering           │    │
│  └────────────────────────────────────────────────────┘    │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

## Components

### kbconfig

The `internal/kbconfig` package parses and validates the YAML
configuration file. Its responsibilities include:

- The package parses YAML using `gopkg.in/yaml.v3`.

- The package loads API keys from separate files referenced by
  `api_key_file` settings.

- The package expands `~` paths to the user's home directory.

- The package applies sensible defaults so the most common
  configurations stay concise.

- The package validates required fields and rejects configurations
  with no enabled embedding provider.

### kbsource

The `internal/kbsource` package fetches documentation from sources.
Its responsibilities include:

- The package supports Git repositories with branch or tag selection
  through `exec.Command`.

- The package supports local filesystem paths with optional
  `doc_path` subdirectories.

- The package sanitises project names so generated working
  directories are safe on every platform.

- The package skips Git `pull` operations when `--skip-updates` is
  set.

### kbconverter

The `internal/kbconverter` package converts each supported document
format to Markdown:

- HTML conversion uses the `JohannesKaufmann/html-to-markdown`
  library; the converter shifts heading levels (H1 → H2) and extracts
  the title from the `<title>` element.

- reStructuredText conversion uses a pattern-based parser that
  recognises both overline+underline and underline-only headings.

- SGML and DocBook XML conversion uses a tag-aware parser that
  recognises the chapter, sect1–sect4, and refsect1–refsect2
  hierarchy used in PostgreSQL documentation.

- Markdown passes through with title extraction from the first H1
  heading.

Every converter returns the Markdown body, the extracted title, and
an error.

### kbchunker

The `internal/kbchunker` package chunks documents with structural
preservation. The package files split the work as follows:

- The `chunker.go` file implements the main chunking algorithm with
  heading-hierarchy tracking.

- The `elements.go` file detects structural elements (code blocks,
  tables, lists, blockquotes, paragraphs).

- The `merge.go` file implements the second pass that merges
  undersized chunks.

The chunking algorithm uses a two-pass approach.

#### Pass 1 — Semantic Boundary Splitting

The first pass walks the document and emits chunks that respect
structural element boundaries:

1. The chunker parses the content into structural elements.

2. The chunker never splits within a structural element.

3. The chunker emits a chunk at an element boundary once the target
   size (250 words) is reached.

4. The chunker uses type-specific splitters for oversized elements:
   paragraphs split at sentence boundaries, code blocks split at line
   boundaries (re-adding fence markers), tables split at row
   boundaries (preserving the header), lists split at top-level item
   boundaries, and blockquotes split at line boundaries.

#### Pass 2 — Merge Undersized Chunks

The second pass joins small chunks with neighbours to avoid orphan
fragments that hurt retrieval quality:

1. The pass identifies chunks below the minimum size (100 words).

2. The pass merges each undersized chunk with the next chunk when
   the combined size stays within limits.

3. The pass prefers forward merging for reading-flow continuity and
   handles trailing undersized chunks by merging backwards.

#### Size Constraints

The chunker enforces the following limits:

```go
TargetChunkSize = 250  // Target words per chunk
MaxChunkSize    = 300  // Hard word limit
MaxChunkChars   = 3000 // Hard character limit
MinSize         = 100  // Minimum before merging
OverlapSize     = 50   // Overlap between chunks
```

The hard limits keep chunks compatible with Ollama models such as
`nomic-embed-text`, which accepts up to 8192 tokens.

### kbembed

The `internal/kbembed` package generates embeddings from one or more
providers. The package supports:

- OpenAI, which processes batches of up to 100 texts per request.

- Voyage AI, which processes batches of up to 100 texts per request.

- Ollama, which processes one text at a time.

- Gemini, which processes batches of up to 100 texts per request.

HTTP transport, authentication headers, and exponential-backoff
retries for all providers are delegated to the
`pgedge-go-llm-lib` library; `kbembed` assembles provider-specific
request bodies and decodes the responses.

The package retries transient API errors with exponential backoff
capped at 60 seconds; the `--max-retries` flag controls the budget.
Context-length errors from Ollama bypass retries and instead trigger
text truncation or chunk skipping.

### kbdatabase

The `internal/kbdatabase` package wraps the SQLite database. It uses
the pure-Go `modernc.org/sqlite` driver so the binary stays
statically linked.

The package responsibilities include:

- The package creates the `chunks` table and supporting indexes on
  first run.

- The package serialises embedding vectors as little-endian float32
  BLOBs.

- The package uses transactions for batch inserts.

- The package supports incremental rebuilds through SHA256 checksum
  lookups and cleanup of stale chunks.

The [Output Database Layout](../guide/output-db.md) guide documents
the schema in detail.

### kbtypes

The `internal/kbtypes` package defines shared data types. The two
primary types are:

```go
type Document struct {
    Title, Content string
    SourceContent  []byte
    FilePath       string
    ProjectName    string
    ProjectVersion string
    DocType        DocumentType
}

type Chunk struct {
    Text, Title, Section string
    HeadingPath          []string
    ElementTypes         []string
    ProjectName          string
    ProjectVersion       string
    FilePath             string
    SourceFileChecksum   string
    OpenAIEmbedding      []float32
    VoyageEmbedding      []float32
    OllamaEmbedding      []float32
    GeminiEmbedding      []float32
}
```

## Build Process Walkthrough

The end-to-end build proceeds as follows:

1. The user runs `pgedge-ai-kb-builder --config build.yaml`.

2. `kbconfig` loads the configuration, expands paths, applies
   defaults, loads API keys, and enumerates the enabled provider/model
   targets.

3. `kbsource.FetchAll` fetches every source once; Git sources clone or
   pull, local sources scan the configured directory.

4. The `main` function then builds one database per target. For each
   target it opens that provider's SQLite database (creating the schema
   on first run) and re-runs the chunking pipeline against the
   already-fetched sources.

5. For each source, the per-target loop walks the directory, filters to
   supported file types, and dispatches each file through the
   pipeline:

    - The pipeline reads the file and computes a SHA256 checksum.

    - The pipeline asks the database whether the file has already
      been processed for this project and version; if so, the
      pipeline skips it.

    - The pipeline checks whether another project version already
      has chunks for this checksum; if so, the pipeline clones the
      existing chunks under the new project version (deduplication).

    - Otherwise, the pipeline converts the file to Markdown and
      chunks the result.

6. `kbdatabase.InsertChunks` writes the target's chunks in batched
   transactions, then `kbembed.GenerateEmbeddings` produces vectors for
   that one provider and persists them.

7. The main function prints per-target summary statistics and exits.

## Error Handling

The build classifies errors as follows:

- Non-fatal errors (unsupported file, conversion failure for a
  single file) log a warning and continue with the next file.

- Fatal errors (missing API key, network error contacting the
  database, configuration validation failure) abort the build with
  a non-zero exit code.

- Transient API errors (HTTP 429, HTTP 5xx) retry within the
  configured budget before being treated as fatal.

The release workflow uses `--max-retries 50` to absorb temporary
rate limits.

## Performance Characteristics

The following measurements describe typical builds (measured during
development; expect variation by region and source size):

- PostgreSQL 17 documentation (≈3,000 pages):

    - Chunks created: ≈30,000

    - Embedding time (OpenAI): 5–10 minutes

    - Database size: ≈250 MB

- Multiple PostgreSQL versions (14–17):

    - Chunks created: ≈150,000

    - Embedding time (OpenAI): 25–50 minutes

    - Database size: ≈500 MB

## Extending the Builder

### Adding a New Document Format

To add a new format:

1. Add format detection in `kbconverter.DetectDocumentType`.

2. Implement a converter function with the signature
   `func(content []byte) (markdown string, title string, err error)`.

3. Add the new branch to `kbconverter.Convert`.

4. Register the new extensions in `kbconverter.GetSupportedExtensions`.

5. Add unit tests with representative sample documents.

### Adding a New Embedding Provider

To add a new provider:

1. Add a configuration struct to `kbconfig.EmbeddingConfig` and
   wire defaults in `applyDefaults`.

2. Implement provider-specific generation in
   `kbembed.EmbeddingGenerator`.

3. Add a new BLOB column to the database schema and update the
   reader/writer code in `kbdatabase`.

4. Add the new embedding field to `kbtypes.Chunk`.

5. Add unit tests using `httptest.NewServer` to mock the provider.

### Adjusting Chunking Behaviour

Tune the chunker by editing the constants in
`internal/kbchunker/chunker.go`:

```go
const (
    TargetChunkSize = 250
    MaxChunkSize    = 300
    MaxChunkChars   = 3000
    OverlapSize     = 50
)
```

`MaxChunkSize` and `MaxChunkChars` are hard limits for Ollama
compatibility; do not raise them without verifying every supported
model.

## See Also

- [Development Setup](development.md) describes the local toolchain.

- [Testing](testing.md) covers the test suite layout.

- [Output Database Layout](../guide/output-db.md) documents the
  SQLite schema in detail.
