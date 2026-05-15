# Building a Knowledgebase

This guide walks through producing a complete knowledgebase end-to-end:
preparing source documentation, configuring the builder, running a
build, and rebuilding incrementally as documentation changes.

For an abbreviated walkthrough, see the [Quick Start](quickstart.md).

## Preparing Domain Documentation

Domain documentation describes the schemas, business rules, and query
patterns relevant to the downstream consumers of the knowledgebase.
You write the documentation as Markdown files (or any other supported
format) and store them in a dedicated directory.

### Writing Schema Documentation

Schema documentation describes each table, its columns, and the data
the table contains. Create one Markdown file per table group or major
topic.

In the following example, the Markdown file documents an e-commerce
database schema with tables and common queries:

```markdown
# E-Commerce Database Schema

## Orders Table

The `orders` table stores all customer purchase records.

| Column        | Type           | Description                       |
|---------------|----------------|-----------------------------------|
| id            | SERIAL         | The unique order identifier.      |
| customer_id   | INTEGER        | References the customers table.   |
| status        | VARCHAR(20)    | The order status value.           |
| total_amount  | NUMERIC(10,2)  | The order total in USD.           |
| created_at    | TIMESTAMPTZ    | The order creation timestamp.     |
```

### Writing Business Rules Documentation

Business rules documentation defines domain terms and metrics that
downstream consumers reference. The following example documents
revenue metrics and customer status definitions:

```markdown
# Business Rules and Glossary

## Revenue Metrics

Net revenue equals the sum of order amounts excluding cancelled
and refunded orders. Gross revenue equals the sum of all order
amounts including cancelled orders. Average order value (AOV)
equals net revenue divided by the count of completed orders.

## Status Definitions

An active customer has at least one order in the last 90 days.
A churned customer has no orders in the last 180 days.
```

### Writing Relationship Documentation

Relationship documentation describes how tables connect through
foreign keys and join patterns:

```markdown
# Table Relationships

## Customer to Orders (One-to-Many)

Each customer can have many orders. Join customers to their orders
using the `customer_id` column.

SELECT c.name, o.id, o.total_amount
FROM customers c
JOIN orders o ON o.customer_id = c.id;
```

## Configuring the Build

The builder reads a YAML configuration file that lists documentation
sources and embedding providers. The [Configuration File
Reference](../reference/config.md) lists every field with comments.

In the following example, the configuration combines local domain
documentation with PostgreSQL upstream documentation and produces
OpenAI embeddings:

```yaml
database_path: "my-project-kb.db"
doc_source_path: "doc-source"

sources:
    - local_path: "~/my-project"
      doc_path: "docs"
      project_name: "My E-Commerce App"
      project_version: "1.0"

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

The [Configuring Sources](sources.md) and [Configuring Embedding
Providers](embeddings.md) guides describe each section in detail.

## Running the Build

Run the builder against your configuration file:

```bash
./bin/pgedge-ai-kb-builder --config my-kb-builder.yaml
```

The build pipeline performs the following work:

1. The builder loads the configuration and resolves API key files.

2. The builder fetches each source by cloning or pulling Git
   repositories and scanning local paths.

3. The converter walks every supported file in each source, converts
   it to Markdown, and chunks the content.

4. The embedding generator calls every enabled provider in batch and
   stores the resulting vectors against the chunks.

5. The database writer persists chunks and embeddings to the output
   SQLite database.

The build prints per-file progress lines and ends with summary
statistics:

```text
Total chunks: 31423
Projects:
  - PostgreSQL 17: 28932 chunks
  - My E-Commerce App 1.0: 2491 chunks

✓ Knowledgebase successfully built: my-project-kb.db
```

## Rebuilding Incrementally

The builder supports incremental rebuilds. Subsequent runs reuse
existing chunks for files that have not changed since the previous
build.

### Detecting Changed Files

The builder computes a SHA256 checksum of every input file. If the
checksum matches the recorded checksum for that file and project, the
builder reuses the existing chunks and embeddings. Otherwise, the
builder reprocesses the file and replaces the chunks.

Run the same command to update:

```bash
./bin/pgedge-ai-kb-builder --config my-kb-builder.yaml
```

The builder pulls each Git repository to pick up upstream changes,
then incrementally reprocesses only modified files.

### Skipping Git Updates

Skip the Git pull step during local development with `--skip-updates`:

```bash
./bin/pgedge-ai-kb-builder --config my-kb-builder.yaml --skip-updates
```

The builder uses the already-cloned working tree as-is.

### Refilling Missing Embeddings

Enable a new embedding provider after the initial build, then use
`--add-missing-embeddings` to fill in only the missing vectors:

```bash
./bin/pgedge-ai-kb-builder --config my-kb-builder.yaml \
    --add-missing-embeddings
```

The builder skips chunks that already have vectors for every enabled
provider.

### Clearing Embeddings

Clear vectors for one provider with `--clear-embeddings <provider>`:

```bash
./bin/pgedge-ai-kb-builder --config my-kb-builder.yaml \
    --clear-embeddings openai
```

Follow with `--add-missing-embeddings` to repopulate.

## Best Practices for Domain Documentation

The following practices improve the quality of retrieval against the
knowledgebase:

- Document every table with its purpose and column descriptions.

- Include example queries for common business questions.

- Define business terms and domain jargon in a glossary file.

- Document join patterns between related tables.

- Include sample data to illustrate expected column values.

- Keep the documentation current when the schema changes.

- Use one Markdown file per major topic or table group.

- Include both simple and complex query examples.

- Write clear column descriptions that distinguish similarly named
  columns.

- Document enum values and status codes with their meanings.

## See Also

- [Configuring Sources](sources.md) describes every source option.

- [Configuring Embedding Providers](embeddings.md) covers each
  provider in detail.

- [Output Database Layout](output-db.md) documents the SQLite schema.

- [Troubleshooting](troubleshooting.md) collects fixes for common
  build issues.
