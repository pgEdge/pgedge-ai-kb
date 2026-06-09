# Release Notes

## Unreleased

- The builder now writes one database per enabled provider/model
  instead of a single combined `kb.db`, keeping each release asset and
  package under GitHub's 2 GiB limit. Each file is named
  `kb-<provider>-<model>.db` (for example,
  `kb-openai-text-embedding-3-small.db`) and uses the same schema, with
  only that provider's embedding column populated. The release pipeline
  publishes each database as its own asset and ships one co-installable
  OS package per provider/model, named
  `pgedge-ai-kb-<provider>-<model>`, that installs its database under
  `/usr/share/pgedge/pgedge-ai-kb/`. Consumers download or install the
  database matching their configured provider.

- The builder now bounds embedding requests with two configurable
  timeouts, `request_timeout` (overall ceiling including retries) and
  `per_attempt_timeout` (per HTTP attempt), under the `embeddings`
  block. A stalled attempt is retried rather than cancelling the whole
  request, which fixes Gemini batches silently dropping chunks when a
  single slow `batchEmbedContents` call exhausted the timeout budget.
  The values default to `10m` and `90s` respectively.

- The pgEdge AI Knowledgebase Builder now supports Gemini as a fourth
  embedding provider; the default model is `gemini-embedding-001`.
  Configure it under the `embeddings.gemini` block in the YAML
  configuration file. The example configuration now enables Gemini by
  default, so it requires a Gemini API key unless you disable the
  provider.

- The embedding HTTP and retry logic has migrated to the shared
  `github.com/pgEdge/pgedge-go-llm-lib` library. This consolidates
  request handling, retry/backoff, and Ollama context-overflow
  truncation into one place across pgEdge tooling.

- The `--max-retries 0` flag previously meant "retry indefinitely";
  it now maps to a very large finite cap. Practical behaviour for
  users is unchanged.

- Existing databases gain a new `gemini_embedding` BLOB column on
  first open via an idempotent `ALTER TABLE`. Old rows have NULL in
  this column until the next build runs with Gemini enabled.

- The minimum required Go toolchain has bumped to 1.26.1 to match
  the shared LLM library.

- The pgEdge AI Knowledgebase Builder is extracted from the pgEdge
  Postgres MCP Server into a standalone project. The Go module is
  `github.com/pgEdge/pgedge-ai-kb` and the binary is renamed from
  `pgedge-nla-kb-builder` to `pgedge-ai-kb-builder`.

- The default output database filename changes from
  `pgedge-nla-kb.db` to `pgedge-ai-kb.db`. Downstream consumers that
  reference the file path by name need updates; consumers that load
  a configured path are unaffected.

- The default configuration filename loaded from the binary directory
  changes from `pgedge-nla-kb-builder.yaml` to
  `pgedge-ai-kb-builder.yaml`.

- The published GoReleaser archives are renamed from
  `pgedge-postgres-mcp-kb-builder_*` to `pgedge-ai-kb-builder_*`.

- Linter findings inherited from the original code base (capitalised
  error strings, tagged-switch suggestion, De Morgan transform) are
  cleaned up so the new project's CI runs `golangci-lint run` on the
  whole codebase without exclusions.

- When an embedding batch fails, the error now identifies the batch
  range (e.g., "OpenAI embed batch 1-100") rather than the specific
  failing chunk. The per-chunk diagnostic block from the previous
  Ollama implementation is no longer available because the shared LLM
  library returns one error per batch of inputs.

## Earlier history

For release notes covering the period when this code lived inside
[pgedge-postgres-mcp](https://github.com/pgEdge/pgedge-postgres-mcp),
see that project's
[changelog](https://github.com/pgEdge/pgedge-postgres-mcp/blob/main/docs/changelog.md).
