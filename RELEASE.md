# Release Process

This document describes how to create releases for the pgEdge AI
Knowledgebase Builder using GoReleaser.

## Prerequisites

- Go 1.25 or higher
- GoReleaser installed (`go install github.com/goreleaser/v2/cmd/goreleaser@latest`)
- GitHub token with repo write permissions (for creating releases)

## Release Artifacts

Each release publishes the following artifacts.

### 1. KB Builder Binaries (`pgedge-ai-kb-builder_*`)

Architecture-specific archives are produced for:

- Linux: amd64, arm64
- macOS: amd64, arm64
- Windows: amd64

Each archive contains:

- `pgedge-ai-kb-builder` binary
- `README.md` and `LICENSE.md`
- `examples/pgedge-ai-kb-builder.yaml` configuration template

### 2. Knowledgebase Database (`kb.db`)

A separate workflow (`release-kb.yml`) produces a pre-built `kb.db`
file containing all sources defined in
`examples/pgedge-ai-kb-builder.yaml`. Downstream consumers (for example,
the pgEdge Postgres MCP Server) download this artifact at image-build
time so they ship with an up-to-date knowledgebase.

## Creating a Binary Release

### 1. Prepare the Release

Review the changelog and update `docs/changelog.md` with notable
changes since the last release:

```bash
git log $(git describe --tags --abbrev=0)..HEAD --oneline
```

### 2. Tag and Push

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

### 3. Automated Release

The `release.yml` workflow runs on tag push. It:

1. Sets up Go on amd64 and arm64 runners.
2. Runs unit tests.
3. Executes GoReleaser to produce per-platform archives.
4. Generates a checksums file.
5. Creates a GitHub Release with auto-generated notes.

## Building the Knowledgebase Database

The `release-kb.yml` workflow is triggered manually
(`workflow_dispatch`) and runs on the `ollama` self-hosted runner.
It:

1. Checks out the requested branch.
2. Builds `pgedge-ai-kb-builder` with `CGO_ENABLED=0`.
3. Starts a local Ollama instance and pulls `nomic-embed-text`.
4. Loads OpenAI and Voyage API keys from secrets.
5. Runs `pgedge-ai-kb-builder` against the canonical
   `examples/pgedge-ai-kb-builder.yaml`.
6. Publishes the resulting `kb.db` as a tagged release.

Run it from the GitHub Actions tab and pass:

- `branch` - the source branch (default `main`).
- `release_tag` - the tag for the published `kb.db`
  (e.g. `kb-2026-05-15`).

## Version Numbering

Follow semantic versioning (semver):

- **Major** (v1.0.0 → v2.0.0): Breaking configuration or schema changes.
- **Minor** (v1.0.0 → v1.1.0): New features, backwards compatible.
- **Patch** (v1.0.0 → v1.0.1): Bug fixes, backwards compatible.

## Changelog Format

Use conventional commit prefixes so release notes are easy to read:

```text
feat: Support per-source max_versions cutoff
fix:  Retry 429 responses from Voyage AI
sec:  Update dependencies to address CVE-2026-xxxxx
docs: Update embedding provider matrix
test: Add integration test for SGML conversion
chore: Update CI Go version
```

## Troubleshooting

### Build Fails in CI

- Check the GitHub Actions logs for the failing job.
- Verify `make test` and `make lint` pass locally.

### GoReleaser Errors

```bash
goreleaser check
goreleaser release --snapshot --clean --config .goreleaser-amd64.yaml
```

### Knowledgebase Build Fails

- Confirm the Ollama runner is online and the `nomic-embed-text`
  model is available.
- Confirm `OPENAI_API_KEY` and `VOYAGE_API_KEY` secrets are populated.
- Use `--max-retries 0` to retry indefinitely (the workflow timeout
  bounds the run).

## Related Documentation

- [GoReleaser Documentation](https://goreleaser.com/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
