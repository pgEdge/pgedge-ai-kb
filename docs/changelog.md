# Release Notes

## Unreleased

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

## Earlier history

For release notes covering the period when this code lived inside
[pgedge-postgres-mcp](https://github.com/pgEdge/pgedge-postgres-mcp),
see that project's
[changelog](https://github.com/pgEdge/pgedge-postgres-mcp/blob/main/docs/changelog.md).
