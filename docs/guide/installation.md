# Installation

The pgEdge AI Knowledgebase Builder ships as a single statically linked
Go binary. You can install it from source or download a pre-built
release archive.

## Prerequisites

The builder runs on Linux, macOS, and Windows. The build itself uses
pure Go, so the host machine needs only the tools below; the resulting
binary has no system dependencies at runtime:

- Go 1.25 or later for source builds.

- Git, for fetching documentation source repositories.

- A POSIX shell and the GNU make utility for running the project's
  Makefile targets.

You also need credentials for at least one embedding provider. The
[Configuring Embedding Providers](embeddings.md) guide lists the
supported options.

## Install From Source

Source installs work well when you want the latest changes or plan to
contribute. Clone the repository and run the default `make` target:

```bash
git clone https://github.com/pgEdge/pgedge-ai-kb.git
cd pgedge-ai-kb
make build
```

The build writes the binary to `bin/pgedge-ai-kb-builder`. Use the
`install` target to copy it to your Go bin directory:

```bash
make install
```

Confirm the install with:

```bash
pgedge-ai-kb-builder --help
```

## Install From a Release Archive

Each tagged release publishes per-platform archives on the
[GitHub Releases page](https://github.com/pgEdge/pgedge-ai-kb/releases).
Download the archive for your operating system and architecture, then
extract it:

```bash
curl -L -o kb-builder.tar.gz \
    https://github.com/pgEdge/pgedge-ai-kb/releases/download/<tag>/<archive>
tar -xzf kb-builder.tar.gz
cd <extracted-directory>
./pgedge-ai-kb-builder --help
```

Each archive ships with `README.md`, `LICENSE.md`, and the example
configuration file. Move the binary onto your `$PATH` to use it from
anywhere on the system.

## Cross-Compile From Source

The Makefile includes targets that cross-compile for every supported
platform. Use these targets to produce all platform binaries in one
pass:

```bash
make build-all
```

The binaries land under `bin/` with platform suffixes.

## Verify the Installation

A successful install prints the help text:

```bash
pgedge-ai-kb-builder --help
```

The help output lists the supported command-line flags. If the
command is not found, confirm that your Go bin directory or the
extracted archive directory is on `$PATH`.
