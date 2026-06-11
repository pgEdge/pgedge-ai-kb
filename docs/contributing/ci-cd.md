# CI/CD

The project's automation lives entirely in GitHub Actions. The
workflows under `.github/workflows/` cover continuous integration,
release publishing, and source drift detection.

## CI Workflows

The following workflows run on every push and pull request to `main`
and `develop`:

- The `ci-kb-builder.yml` workflow runs `go mod verify`, `golangci-lint
  run`, `go vet`, `gofmt -l`, `go build`, and `go test -race` for
  every Go package. It uploads the built binary and a coverage report
  as workflow artifacts.

- The `ci-docs.yml` workflow builds the MkDocs site with the
  `requirements.txt` Python pin. It uploads the rendered site as an
  artifact.

Both workflows pin Go to 1.25 and golangci-lint to v2.5.0 to match
local development.

## Release Workflows

Two release workflows publish artifacts.

### Binary Release (`release.yml`)

The workflow triggers on Git tags matching `v*` and runs the
following stages:

1. The `build-amd64` job runs `go test` and `goreleaser release
   --config .goreleaser-amd64.yaml` on a native x86_64 runner.

2. The `build-arm64` job runs the same commands on a native arm64
   runner with the arm64 GoReleaser configuration.

3. The `release` job downloads the per-architecture artifacts,
   produces a SHA256 checksum file, and publishes a GitHub Release.

The workflow uses native-architecture runners (no QEMU emulation),
which keeps the build fast and reliable for both architectures.

### Knowledgebase Release (`release-kb.yml`)

The workflow builds a fresh set of `kb-<provider>-<model>.db` files,
one per enabled provider, and publishes them as a GitHub Release. The
workflow is triggered manually (`workflow_dispatch`) and runs on the
`ollama` self-hosted runner.

The workflow:

1. Validates that the requested branch is either `main` or
   `release/...` (defence against running untrusted refs with
   secret access).

2. Starts a local Ollama instance and pulls `nomic-embed-text`.

3. Builds `pgedge-ai-kb-builder` with `CGO_ENABLED=0`.

4. Writes OpenAI, Voyage, and Gemini API keys from secrets into the
   runner's ephemeral directory with mode `0600`.

5. Rewrites the example configuration to point at the temporary key
   files and runs the builder with `--max-retries 50`.

6. Publishes the resulting `kb-<provider>-<model>.db` files as a tagged
   release named after the `release_tag` input (for example
   `kb-2026-05-15`).

Run the workflow from the GitHub Actions tab. Provide a `branch`
(default `main`) and a `release_tag` (for example
`kb-2026-05-15`).

## Source Drift Detection (`check-sources.yml`)

The workflow compares `examples/pgedge-ai-kb-builder.yaml` against the
canonical `sources.yaml` in the `pgedge-doc-sources` repository. It
runs on manual dispatch (the schedule line is commented out by
default) and:

1. Fetches the SSOT `sources.yaml` using the
   `PGEDGE_BUILDER_TOKEN` secret.

2. Detects missing entries, SSH URLs that need conversion to HTTPS,
   and versioned entries that have been removed from the SSOT.

3. Generates a fix patch on the `auto/sync-kb-builder` branch and
   opens or updates a pull request when actionable drift exists.

The workflow never comments on developer pull requests. Consumer
entries that the SSOT does not expose to kb-builder — whether removed
outright or held back by policy — are handled by kind:

- Versioned entries (those carrying a `project_version`) that no
  longer exist in the SSOT are auto-removed — for example, a patch
  release that the SSOT has bumped past.

- Living/unversioned sources (branch refs with no `project_version`,
  such as `branch: main` self-references) are reported but never
  removed.

- Policy-excluded versions (still in the SSOT but beyond a
  component's `max_versions` cutoff) are reported but never removed.

## Secrets

The release and drift workflows reference the following repository
secrets:

- The `OPENAI_API_KEY` secret holds the API key used by
  `release-kb.yml` to generate OpenAI embeddings.

- The `VOYAGE_API_KEY` secret holds the API key used by
  `release-kb.yml` to generate Voyage embeddings.

- The `PGEDGE_BUILDER_TOKEN` secret holds a personal access token
  with permission to clone private source repositories and open pull
  requests in this repository.

The `GITHUB_TOKEN` secret is automatically provided by GitHub.

## Runner Labels

The release-kb workflow runs on the self-hosted `ollama` runner. The
runner label declaration lives in `.github/actionlint.yaml` so that
`actionlint` recognises the custom label.

## See Also

- [Development Setup](development.md) covers local development.

- [Testing](testing.md) explains how the test suite is organised.

- The [`RELEASE.md`](https://github.com/pgEdge/pgedge-ai-kb/blob/main/RELEASE.md)
  file in the repository root describes the release process from a
  release manager's perspective.
