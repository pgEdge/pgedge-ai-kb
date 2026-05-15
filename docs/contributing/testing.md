# Testing

The project ships a unit-test suite that covers every internal
package. The test suite runs entirely offline; no external network
or API services are required.

## Running the Suite

The default `make test` target runs every unit test:

```bash
make test
```

The `make test-coverage` target produces a coverage profile and prints
per-function coverage:

```bash
make test-coverage
go tool cover -html=coverage.out -o coverage.html
```

Open `coverage.html` to inspect untested code paths.

## Test Layout

Each internal package keeps its tests alongside the production code:

- The `internal/kbchunker/chunker_test.go`,
  `internal/kbchunker/elements_test.go`, and
  `internal/kbchunker/merge_test.go` files cover the chunker.

- The `internal/kbconfig/config_test.go` file covers configuration
  parsing, validation, and path expansion.

- The `internal/kbconverter/converter_test.go` file covers the
  per-format converters with embedded sample documents.

- The `internal/kbdatabase/database_test.go` file covers SQLite
  schema creation and CRUD against an in-memory database.

- The `internal/kbembed/embeddings_test.go` file covers the
  embedding providers using mocked HTTP servers.

- The `internal/kbsource/source_test.go` file covers Git and local
  source fetching; tests use temporary directories and never touch
  the user's home directory.

## Writing New Tests

Follow the patterns in the existing test files. The most common
conventions are:

- Each test uses subtests (`t.Run`) so the suite reports per-case
  results.

- Tests that touch the filesystem use `t.TempDir()` to create a
  cleanly removed scratch directory.

- Tests that exercise HTTP behaviour spin up an `httptest.Server`
  and point the code under test at the server URL.

- Tests avoid network access; the embedded providers mock the
  external APIs.

## CI Behaviour

The `ci-kb-builder.yml` workflow runs the full test suite on every
push and pull request. The same workflow runs `go vet`, formats
checking with `gofmt -l`, and `golangci-lint run`. Pull requests
cannot merge while any of these checks fail.

The release workflow (`release.yml`) re-runs the test suite on both
amd64 and arm64 runners before producing a GoReleaser build.

## See Also

- [Development Setup](development.md) covers local development.

- [CI/CD](ci-cd.md) documents the automation pipelines.
