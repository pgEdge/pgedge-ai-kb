#!/usr/bin/env bash
#
# DEB build for pgedge-postgres-mcp-kb.
#
# Invoked by common/build.sh from pgedge-builder-action via
#   prepare → build → post_build.
# common.sh has already been sourced; helpers come from
# common/common-functions.sh.

set -euo pipefail

BUILD_DIR="/tmp/pgedge_kb_deb_build"
SRC_DIR="${BUILD_DIR}/src"
CWD="$(pwd)"

export DEBIAN_FRONTEND=noninteractive
ARCH="$(uname -m)"
if [ "$ARCH" = "aarch64" ]; then
    ARCH="arm64"
fi

# Staged artifact paths — see build-rpm.sh for the rationale. The workflow
# downloads both the kb.db (from KB_DB_RELEASE_TAG GH release) and the binary
# tarball (from the build-amd64 / build-arm64 jobs in the SAME workflow run)
# into $GITHUB_WORKSPACE before this script runs; the container sees them
# under /build.
KB_DB_HOST_PATH="${KB_DB_HOST_PATH:-/build/pkg/kb.db.cached/kb.db}"
BINARY_TARBALL_DIR="${BINARY_TARBALL_DIR:-/build/pkg/binary.cached}"

prepare() {
    setup_apt_build_env
    apt-get install -y jq sqlite3

    # Reconstruct goreleaser's .Version (= the tag minus leading 'v'). Same
    # logic as build-rpm.sh, except common.sh has already converted '_' to
    # '~' for apt — so the iteration separator we strip is '~'.
    if [[ "${PGEDGE_KB_BUILDNUM}" == *~* ]]; then
        export BUILDER_VERSION="${PGEDGE_KB_VERSION}-${PGEDGE_KB_BUILDNUM%~*}"
    else
        export BUILDER_VERSION="${PGEDGE_KB_VERSION}"
    fi

    local tarball="pgedge-ai-kb-builder_${BUILDER_VERSION}_linux_${ARCH}.tar.gz"
    local tarball_path="${BINARY_TARBALL_DIR}/${tarball}"

    if [ ! -f "${tarball_path}" ]; then
        echo "::error::Binary tarball not found at ${tarball_path}"
        ls -la "${BINARY_TARBALL_DIR}" 2>/dev/null || echo "  (missing)"
        exit 1
    fi
    echo "Staged binary tarball: ${tarball}"

    if [ ! -f "${KB_DB_HOST_PATH}" ]; then
        echo "::error::kb.db not found at ${KB_DB_HOST_PATH}"
        echo "validate-kb-db is expected to fetch it from ${KB_DB_RELEASE_TAG} before this cell runs."
        exit 1
    fi

    echo "Setting up source directory at ${SRC_DIR}..."
    rm -rf "${SRC_DIR}"
    mkdir -p "${SRC_DIR}/builder"

    echo "Extracting binary tarball..."
    tar -xzf "${tarball_path}" -C "${SRC_DIR}/builder"

    echo "Staging kb.db..."
    cp "${KB_DB_HOST_PATH}" "${SRC_DIR}/kb.db"

    echo "Copying debian/ tree..."
    cp -r "${CWD}/${COMPONENT_NAME}/deb/debian" "${SRC_DIR}/"

    echo "Staging example config..."
    cp "${CWD}/${COMPONENT_NAME}/common/pgedge-ai-kb-builder.yaml" \
        "${SRC_DIR}/debian/"

    echo "Installing build deps..."
    cd "${SRC_DIR}"
    sudo apt-get update
    sudo apt-get build-dep -y .
}

build() {
    cd "${SRC_DIR}"
    local distro
    distro="$(lsb_release -cs)"

    cat > debian/changelog <<EOF
pgedge-postgres-mcp-kb (${PGEDGE_KB_VERSION}-${PGEDGE_KB_BUILDNUM}.${distro}) ${distro}; urgency=medium

  * Automated build from ${GITHUB_REF_NAME:-local}
  * Bundles kb-builder + kb.db from ${KB_DB_RELEASE_TAG}

 -- Muhammad Aqeel <muhammad.aqeel@pgedge.com>  $(date -R)
EOF

    dpkg-buildpackage -us -uc -b
}

post_build() {
    echo "Copying .deb packages to /output..."
    sudo mkdir -p /output
    rename_ddeb_packages "${BUILD_DIR}"
    sudo cp "${BUILD_DIR}"/*.deb /output/ || echo "No .deb packages found"
}
