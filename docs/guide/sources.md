# Configuring Sources

The builder reads documentation from one or more sources defined in the
`sources` list of its configuration file. Each source describes either
a Git repository or a local directory.

## Source Fields

Every source supports the following fields:

- The `project_name` field is required; it identifies the source in
  the output database and in search filters.

- The `project_version` field is optional; supply it when you intend
  to ingest multiple versions of the same project.

- The `doc_path` field is optional; it points to a subdirectory
  inside the source that contains documentation. When omitted, the
  builder processes the entire source.

Each source must include exactly one of `git_url` or `local_path`. You
cannot specify both for the same source.

## Git Repository Sources

A Git source clones a repository, checks out a branch or a tag, and
processes the documentation in `doc_path`. Specify either `branch` or
`tag`, not both:

```yaml
sources:
    - git_url: "https://github.com/postgres/postgres.git"
      tag: "REL_17_4"
      doc_path: "doc/src/sgml"
      project_name: "PostgreSQL"
      project_version: "17"
```

The builder caches each clone under `doc_source_path` and reuses it on
subsequent runs. Use the `--skip-updates` flag to skip the `git pull`
step during local development.

### Multiple Versions of One Project

Add multiple sources with the same `project_name` but different
`project_version` values to ingest several versions of one project:

```yaml
sources:
    - git_url: "https://github.com/postgres/postgres.git"
      tag: "REL_17_4"
      doc_path: "doc/src/sgml"
      project_name: "PostgreSQL"
      project_version: "17"

    - git_url: "https://github.com/postgres/postgres.git"
      tag: "REL_16_8"
      doc_path: "doc/src/sgml"
      project_name: "PostgreSQL"
      project_version: "16"
```

The builder stores chunks with the appropriate `project_version` for
each source so downstream consumers can filter on it.

### Cross-Version Deduplication

When two versions of a project contain identical file content (matching
SHA256 checksum), the builder reuses chunks and embeddings from the
already-processed copy and tags them with the new project version. This
deduplication avoids duplicated embedding work across closely related
release branches.

## Local Path Sources

A local source ingests documentation directly from the filesystem.
This is the standard pattern for indexing internal documentation that
is not committed to a repository accessible by the build host:

```yaml
sources:
    - local_path: "~/projects/my-app"
      doc_path: "docs"
      project_name: "My App"
      project_version: "1.0"
```

The path may be absolute or use `~` to refer to the user's home
directory. The builder leaves local sources untouched; it never writes
back to the source directory.

## Combining Sources

A single build can mix Git and local sources. The example below pairs
PostgreSQL upstream documentation with a local product documentation
tree:

```yaml
sources:
    - git_url: "https://github.com/postgres/postgres.git"
      branch: "REL_17_STABLE"
      doc_path: "doc/src/sgml"
      project_name: "PostgreSQL"
      project_version: "17"

    - local_path: "~/projects/my-app"
      doc_path: "docs"
      project_name: "My App"
      project_version: "1.0"
```

Downstream consumers can search across all sources or restrict the
search to a single `project_name` and `project_version` pair.

## Supported File Formats

The builder recognises files by extension and converts each one to
Markdown before chunking. The following formats are supported:

- The Markdown format (`.md`) passes through with title extraction.

- The HTML formats (`.html`, `.htm`) convert with the
  `html-to-markdown` library.

- The reStructuredText format (`.rst`) converts through a built-in
  pattern-based parser.

- The SGML formats (`.sgml`, `.sgm`) convert through a DocBook-aware
  pattern parser; the parser supports the PostgreSQL documentation
  source format.

- The DocBook XML format (`.xml`) converts through the same parser as
  SGML.

Files with other extensions are skipped silently. Inspect builder logs
to see which files the build processes.
