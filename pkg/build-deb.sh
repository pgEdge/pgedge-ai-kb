#!/usr/bin/env bash
#
# DEB build for pgedge-ai-kb (data-only, Architecture: all).
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

# kb.db is staged into the container by the workflow before
# pgedge-builder-action runs. The host's pkg/kb.db.cached/ maps to
# /build/pkg/kb.db.cached/ in the container.
#
# This package doesn't ship the kb-builder binary or example config —
# those live elsewhere (GoReleaser tarballs and the pgedge-postgres-mcp
# server package, respectively).
KB_DB_HOST_PATH="${KB_DB_HOST_PATH:-/build/pkg/kb.db.cached/kb.db}"

prepare() {
    setup_apt_build_env
    apt-get install -y jq sqlite3

    if [ ! -f "${KB_DB_HOST_PATH}" ]; then
        echo "::error::kb.db not found at ${KB_DB_HOST_PATH}"
        echo "validate-kb-db is expected to stage it before this cell runs."
        exit 1
    fi

    # Read the kb release tag from the artifact's sidecar file. See
    # build-rpm.sh for rationale.
    local release_tag_file
    release_tag_file="$(dirname "${KB_DB_HOST_PATH}")/RELEASE_TAG"
    if [ -f "${release_tag_file}" ]; then
        export KB_DB_RELEASE_TAG="$(cat "${release_tag_file}")"
        echo "kb.db release tag: ${KB_DB_RELEASE_TAG}"
    else
        echo "::warning::RELEASE_TAG file missing alongside kb.db"
    fi

    # Clear the entire build workspace, not just SRC_DIR. dpkg-buildpackage
    # drops .deb files directly under BUILD_DIR (the parent of the source
    # tree); if a previous run left .debs there, post_build() would copy
    # those stale artifacts to /output. In CI each container is fresh so
    # this is defensive, but local reruns hit it.
    echo "Resetting build workspace at ${BUILD_DIR}..."
    rm -rf "${BUILD_DIR}"
    mkdir -p "${SRC_DIR}"

    echo "Staging kb.db..."
    cp "${KB_DB_HOST_PATH}" "${SRC_DIR}/kb.db"

    echo "Copying license + readme..."
    cp "${CWD}/LICENSE.md" "${SRC_DIR}/LICENSE.md"
    cp "${CWD}/README.md" "${SRC_DIR}/README.md"

    echo "Copying debian/ tree..."
    cp -r "${CWD}/${COMPONENT_NAME}/deb/debian" "${SRC_DIR}/"

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
pgedge-ai-kb (${PGEDGE_KB_VERSION}-${PGEDGE_KB_BUILDNUM}.${distro}) ${distro}; urgency=medium

  * Automated build from ${GITHUB_REF_NAME:-local}
  * kb.db from ${KB_DB_RELEASE_TAG}

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
