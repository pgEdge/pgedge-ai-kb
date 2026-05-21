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

    local tarball="pgedge-ai-kb-builder_${BINARY_VERSION}_linux_${ARCH}.tar.gz"
    local tarball_path="${BINARY_TARBALL_DIR}/${tarball}"

    echo "Verifying staged binary tarball at ${tarball_path}..."
    if [ ! -f "${tarball_path}" ]; then
        echo "Error: binary tarball not found at ${tarball_path}"
        echo "The build-amd64 / build-arm64 jobs are expected to produce"
        echo "release-artifacts-<arch>, and the workflow must download that"
        echo "artifact into pkg/binary.cached/ before this cell runs."
        echo "Contents of ${BINARY_TARBALL_DIR}:"
        ls -la "${BINARY_TARBALL_DIR}" 2>/dev/null || echo "  (missing)"
        exit 1
    fi

    echo "Verifying staged kb.db at ${KB_DB_HOST_PATH}..."
    if [ ! -f "${KB_DB_HOST_PATH}" ]; then
        echo "Error: kb.db not found at ${KB_DB_HOST_PATH}"
        echo "The validate-kb-db job is expected to fetch it from the"
        echo "${KB_DB_RELEASE_TAG} GitHub release before this cell runs."
        exit 1
    fi

    echo "Setting up source directory at ${SRC_DIR}..."
    rm -rf "${SRC_DIR}"
    mkdir -p "${SRC_DIR}/builder"

    echo "Extracting binary tarball from current workflow run..."
    tar -xzf "${tarball_path}" -C "${SRC_DIR}/builder"

    echo "Staging kb.db from current workflow run..."
    cp "${KB_DB_HOST_PATH}" "${SRC_DIR}/kb.db"

    echo "Copying debian/ tree..."
    cp -r "${CWD}/${COMPONENT_NAME}/deb/debian" "${SRC_DIR}/"

    echo "Staging example config + VERSION..."
    cp "${CWD}/${COMPONENT_NAME}/common/pgedge-ai-kb-builder.yaml" \
        "${SRC_DIR}/debian/"
    sed \
        -e "s|@@VERSION@@|${PGEDGE_KB_VERSION_DEB}|" \
        -e "s|@@BUILDER_VERSION@@|${BINARY_VERSION}|" \
        -e "s|@@KB_DB_VERSION@@|${KB_DB_VERSION}|" \
        -e "s|@@BUILD_DATE@@|$(date -u +%Y-%m-%dT%H:%M:%SZ)|" \
        -e "s|@@BUILD_COMMIT@@|${GITHUB_SHA:-unknown}|" \
        -e "s|@@REPO_TYPE@@|${REPO_TYPE}|" \
        "${CWD}/${COMPONENT_NAME}/common/VERSION.tmpl" \
        > "${SRC_DIR}/debian/VERSION"

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
pgedge-postgres-mcp-kb (${PGEDGE_KB_VERSION_DEB}.${distro}) ${distro}; urgency=medium

  * Automated build from ${GITHUB_REF_NAME:-local}
  * Bundles kb-builder ${BINARY_VERSION} and kb.db ${KB_DB_VERSION}

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
