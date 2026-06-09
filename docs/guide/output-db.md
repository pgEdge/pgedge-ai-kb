# Output Database Layout

The builder writes one SQLite database per enabled provider/model,
each holding chunks and that provider's embeddings. Downstream
consumers open the relevant file read-only and run vector similarity
queries against the embedding columns.

## File Naming

The builder emits one database per enabled embedding provider, named
for the provider and the model it embeds with:

```text
kb-openai-text-embedding-3-small.db
kb-voyage-voyage-3.db
kb-ollama-nomic-embed-text.db
kb-gemini-gemini-embedding-001.db
```

The builder derives each filename from the configured `database_path`;
it appends `-<provider>-<model>` to the stem. Every file uses the same
schema described below, but only the column for that file's provider is
populated; the other embedding columns remain NULL. A consumer opens
the database matching its configured provider and reads that provider's
column.

## File Format

Each database is a standard SQLite 3 file. You can inspect one with the
`sqlite3` CLI or any SQLite-aware tool:

```bash
sqlite3 kb-openai-text-embedding-3-small.db ".schema"
```

The database has a `chunks` table that holds every chunk produced by
the build and a small number of supporting indexes.

## chunks Table

The `chunks` table holds the text and embeddings for every processed
chunk. The following SQL shows the table schema:

```sql
CREATE TABLE chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    text TEXT NOT NULL,
    title TEXT,
    section TEXT,
    project_name TEXT NOT NULL,
    project_version TEXT NOT NULL,
    file_path TEXT,
    source_file_checksum TEXT,
    openai_embedding BLOB,
    voyage_embedding BLOB,
    ollama_embedding BLOB,
    gemini_embedding BLOB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

The columns serve the following purposes:

- The `text` column stores the chunk's Markdown content.

- The `title` column stores the document title that produced the
  chunk.

- The `section` column stores the most specific heading the chunk
  belongs to.

- The `project_name` and `project_version` columns store the source
  identifiers for filtering at query time.

- The `file_path` column stores the original source file path.

- The `source_file_checksum` column stores the SHA256 of the source
  file; the builder uses it for incremental rebuilds and cross-version
  deduplication.

- The `openai_embedding`, `voyage_embedding`, `ollama_embedding`, and
  `gemini_embedding` columns hold serialised float32 vectors as
  little-endian BLOBs. Each column is populated only when the
  corresponding provider was enabled at build time. The
  `gemini_embedding` column is added to existing databases via an
  idempotent `ALTER TABLE` on first open.

- The `created_at` column records when the chunk was inserted.

## Indexes

The schema includes the following indexes for fast filtering:

```sql
CREATE INDEX idx_project
    ON chunks(project_name, project_version);
CREATE INDEX idx_title ON chunks(title);
CREATE INDEX idx_section ON chunks(section);
CREATE INDEX idx_checksum ON chunks(source_file_checksum);
```

The `idx_project` index supports filtered searches scoped to a single
project or version. The `idx_checksum` index supports the builder's
incremental rebuild logic.

## Embedding Encoding

Embeddings are stored as little-endian byte sequences of `float32`
values:

- Each dimension occupies 4 bytes.

- The total BLOB size equals the embedding dimensionality times 4.

- The byte order is little-endian on every platform; consumers must
  decode with `binary.LittleEndian` (Go) or the equivalent in other
  languages.

The OpenAI `text-embedding-3-small` model produces 1536-dimensional
vectors, so an `openai_embedding` BLOB is 6144 bytes. Voyage `voyage-3`
produces 1024-dimensional vectors (4096 bytes). Ollama
`nomic-embed-text` produces 768-dimensional vectors (3072 bytes). The
Gemini `gemini-embedding-001` model produces 3072-dimensional vectors
(12288 bytes).

## Querying Patterns

Downstream consumers typically:

1. Load every chunk's embedding for the configured provider.

2. Compute cosine similarity (or another distance) between the query
   vector and each chunk's vector.

3. Return the top N chunks above a similarity threshold.

For larger databases consumers can pre-filter on `project_name` and
`project_version` using the index, then compute similarity only on
the filtered subset.

## Vacuuming

After very large incremental updates, the database file may include
unused space. Reclaim it with `VACUUM`:

```bash
sqlite3 kb-openai-text-embedding-3-small.db "VACUUM;"
```

The builder does not run `VACUUM` automatically; you can schedule it
in your release pipeline if database size matters.
