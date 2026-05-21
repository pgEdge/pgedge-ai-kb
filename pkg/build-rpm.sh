#!/bin/bash
#
# RPM build for pgedge-postgres-mcp-kb.
#
# Invoked by common/build.sh from pgedge-builder-action via
#   prepare → build → post_build.
# common.sh has already been sourced; helpers come from
# common/common-functions.sh.

set -euo pipefail

RHEL="$(rpm --eval %rhel)"
ARCH="$(uname -m)"
if [ "$ARCH" = "aarch64" ]; then
    ARCH="arm64"
fi

# Both source artifacts are staged into the container by the workflow before
# pgedge-builder-action runs. Their mount paths follow $GITHUB_WORKSPACE → /build.
#
#   kb.db          — downloaded by the validate-kb-db job from the
#                    KB_DB_RELEASE_TAG GitHub release, then uploaded as a
#                    workflow artifact, then downloaded into each cell at
#                    pkg/kb.db.cached/kb.db.
#   binary tarball — produced earlier in the same workflow run by the
#                    build-amd64 / build-arm64 GoReleaser jobs, downloaded
#                    into each cell at pkg/binary.cached/<filename>.
#
# Neither is fetched from GitHub Releases at packaging time. The cells must
# consume the artifacts the current commit just produced, not whatever
# happens to be published.
KB_DB_HOST_PATH="${KB_DB_HOST_PATH:-/build/pkg/kb.db.cached/kb.db}"
BINARY_TARBALL_DIR="${BINARY_TARBALL_DIR:-/build/pkg/binary.cached}"

prepare() {
    setup_dnf_build_env
    dnf install -y jq sqlite

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

    echo "Copying spec to ~/rpmbuild/SPECS/..."
    cp "${COMPONENT_NAME}/rpm/pgedge-postgres-mcp-kb.spec" \
        ~/rpmbuild/SPECS/

    echo "Staging binary tarball from current workflow run..."
    cp "${tarball_path}" ~/rpmbuild/SOURCES/

    echo "Staging kb.db from current workflow run..."
    cp "${KB_DB_HOST_PATH}" ~/rpmbuild/SOURCES/kb.db

    echo "Staging example config..."
    cp "${COMPONENT_NAME}/common/pgedge-ai-kb-builder.yaml" \
        ~/rpmbuild/SOURCES/

    echo "Rendering VERSION metadata..."
    sed \
        -e "s|@@VERSION@@|${PGEDGE_KB_VERSION}-${PGEDGE_KB_RELEASE}|" \
        -e "s|@@BUILDER_VERSION@@|${BINARY_VERSION}|" \
        -e "s|@@KB_DB_VERSION@@|${KB_DB_VERSION}|" \
        -e "s|@@BUILD_DATE@@|$(date -u +%Y-%m-%dT%H:%M:%SZ)|" \
        -e "s|@@BUILD_COMMIT@@|${GITHUB_SHA:-unknown}|" \
        -e "s|@@REPO_TYPE@@|${REPO_TYPE}|" \
        "${COMPONENT_NAME}/common/VERSION.tmpl" > ~/rpmbuild/SOURCES/VERSION

    echo "Installing build deps from spec..."
    dnf builddep -y \
        --define "pgedge_kb_version ${PGEDGE_KB_VERSION}" \
        --define "pgedge_kb_release ${PGEDGE_KB_RELEASE}" \
        --define "builder_version ${BINARY_VERSION}" \
        --define "arch ${ARCH}" \
        ~/rpmbuild/SPECS/pgedge-postgres-mcp-kb.spec
}

build() {
    echo "Building RPM and SRPM..."
    QA_RPATHS=$(( 0xffff )) rpmbuild -ba \
        ~/rpmbuild/SPECS/pgedge-postgres-mcp-kb.spec \
        --define "pgedge_kb_version ${PGEDGE_KB_VERSION}" \
        --define "pgedge_kb_release ${PGEDGE_KB_RELEASE}" \
        --define "builder_version ${BINARY_VERSION}" \
        --define "arch ${ARCH}"
}

post_build() {
    echo "Copying built RPMs to /output..."
    mkdir -p /output
    cp -v ~/rpmbuild/RPMS/*/*.rpm /output/ || echo "No binary RPMs found"
    cp -v ~/rpmbuild/SRPMS/*.src.rpm /output/ || echo "No SRPM found"

    sign_rpms /output/*.rpm
    validate_signatures /output/*.rpm
}
