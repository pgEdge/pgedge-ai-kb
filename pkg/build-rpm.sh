#!/bin/bash
#
# RPM build for pgedge-ai-kb (data-only, noarch).
#
# Invoked by common/build.sh from pgedge-builder-action via
#   prepare → build → post_build.
# common.sh has already been sourced; helpers come from
# common/common-functions.sh.

set -euo pipefail

# kb.db is staged into the container by the workflow before
# pgedge-builder-action runs. Its mount path follows
# $GITHUB_WORKSPACE → /build, so pkg/kb.db.cached/ on the host
# is /build/pkg/kb.db.cached/ in the container.
#
# This package doesn't ship the kb-builder binary or the example
# config — those live elsewhere (GoReleaser tarballs and the
# pgedge-postgres-mcp server package, respectively).
KB_DB_HOST_PATH="${KB_DB_HOST_PATH:-/build/pkg/kb.db.cached/kb.db}"

prepare() {
    setup_dnf_build_env
    dnf install -y jq sqlite

    if [ ! -f "${KB_DB_HOST_PATH}" ]; then
        echo "::error::kb.db not found at ${KB_DB_HOST_PATH}"
        echo "validate-kb-db is expected to stage it before this cell runs."
        exit 1
    fi

    # Read the kb release tag from the artifact's sidecar file. The
    # generate-kb-db / discover-kb-db jobs write it alongside kb.db.
    # Falls back to "unknown" if missing (kept resilient for local builds).
    local release_tag_file
    release_tag_file="$(dirname "${KB_DB_HOST_PATH}")/RELEASE_TAG"
    if [ -f "${release_tag_file}" ]; then
        export KB_DB_RELEASE_TAG="$(cat "${release_tag_file}")"
        echo "kb.db release tag: ${KB_DB_RELEASE_TAG}"
    else
        echo "::warning::RELEASE_TAG file missing alongside kb.db"
    fi

    echo "Copying spec to ~/rpmbuild/SPECS/..."
    cp "${COMPONENT_NAME}/rpm/pgedge-ai-kb.spec" \
        ~/rpmbuild/SPECS/

    echo "Staging sources..."
    cp "${KB_DB_HOST_PATH}" ~/rpmbuild/SOURCES/kb.db
    cp LICENSE.md ~/rpmbuild/SOURCES/
    cp README.md ~/rpmbuild/SOURCES/

    echo "Installing build deps from spec..."
    dnf builddep -y \
        --define "pgedge_kb_version ${PGEDGE_KB_VERSION}" \
        --define "pgedge_kb_buildnum ${PGEDGE_KB_BUILDNUM}" \
        ~/rpmbuild/SPECS/pgedge-ai-kb.spec
}

build() {
    echo "Building RPM and SRPM..."
    QA_RPATHS=$(( 0xffff )) rpmbuild -ba \
        ~/rpmbuild/SPECS/pgedge-ai-kb.spec \
        --define "pgedge_kb_version ${PGEDGE_KB_VERSION}" \
        --define "pgedge_kb_buildnum ${PGEDGE_KB_BUILDNUM}"
}

post_build() {
    echo "Copying built RPMs to /output..."
    mkdir -p /output
    cp -v ~/rpmbuild/RPMS/*/*.rpm /output/ || echo "No binary RPMs found"
    cp -v ~/rpmbuild/SRPMS/*.src.rpm /output/ || echo "No SRPM found"

    sign_rpms /output/*.rpm
    validate_signatures /output/*.rpm
}
