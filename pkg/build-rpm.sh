#!/bin/bash
#
# RPM build for pgedge-ai-kb (data-only, noarch).
#
# Invoked by common/build.sh from pgedge-builder-action via
#   prepare → build → post_build.
# common.sh has already been sourced; helpers come from
# common/common-functions.sh.

set -euo pipefail

RHEL="$(rpm --eval %rhel)"

# kb.db is staged into the container by the workflow before
# pgedge-builder-action runs. Its mount path follows
# $GITHUB_WORKSPACE → /build, so pkg/kb.db.cached/ on the host
# is /build/pkg/kb.db.cached/ in the container.
#
# This package doesn't ship the kb-builder binary or the example
# config — those live elsewhere (GoReleaser tarballs and the
# pgedge-postgres-mcp server package, respectively).
# Directory the workflow stages the per-provider databases into. Its
# host path pkg/kb.db.cached/ maps to /build/pkg/kb.db.cached/ in the
# container ($GITHUB_WORKSPACE → /build).
KB_DB_CACHE_DIR="${KB_DB_CACHE_DIR:-/build/pkg/kb.db.cached}"

# pkg_suffix_for prints the package suffix for a database filename, e.g.
# kb-openai-text-embedding-3-small.db -> openai-text-embedding-3-small.
pkg_suffix_for() {
    local base; base="$(basename "$1")"
    base="${base#kb-}"
    printf '%s' "${base%.db}"
}

prepare() {
    setup_dnf_build_env
    dnf install -y jq sqlite

    shopt -s nullglob
    local dbs=("${KB_DB_CACHE_DIR}"/kb-*.db)
    if [ "${#dbs[@]}" -eq 0 ]; then
        echo "::error::no kb-*.db found under ${KB_DB_CACHE_DIR}"
        echo "validate-kb-db is expected to stage them before this cell runs."
        exit 1
    fi

    # Read the kb release tag from the artifact's sidecar file. The
    # generate-kb-db / discover-kb-db jobs write it alongside the
    # databases. Falls back to "unknown" if missing (resilient for local
    # builds).
    local release_tag_file="${KB_DB_CACHE_DIR}/RELEASE_TAG"
    if [ -f "${release_tag_file}" ]; then
        KB_DB_RELEASE_TAG="$(cat "${release_tag_file}")"
        export KB_DB_RELEASE_TAG
        echo "kb release tag: ${KB_DB_RELEASE_TAG}"
    else
        echo "::warning::RELEASE_TAG file missing alongside kb-*.db"
    fi

    echo "Copying spec to ~/rpmbuild/SPECS/..."
    cp "${COMPONENT_NAME}/rpm/pgedge-ai-kb.spec" ~/rpmbuild/SPECS/

    echo "Staging sources..."
    cp "${dbs[@]}" ~/rpmbuild/SOURCES/
    cp LICENSE.md README.md ~/rpmbuild/SOURCES/

    # builddep needs the pkg_suffix/db_filename defines to parse the
    # spec; the deps are identical across databases, so use the first.
    local first db_filename pkg_suffix
    first="$(basename "${dbs[0]}")"
    db_filename="${first}"
    pkg_suffix="$(pkg_suffix_for "${first}")"
    echo "Installing build deps from spec..."
    dnf builddep -y \
        --define "pgedge_kb_version ${PGEDGE_KB_VERSION}" \
        --define "pgedge_kb_buildnum ${PGEDGE_KB_BUILDNUM}" \
        --define "pkg_suffix ${pkg_suffix}" \
        --define "db_filename ${db_filename}" \
        ~/rpmbuild/SPECS/pgedge-ai-kb.spec
}

build() {
    echo "Building one RPM (and SRPM) per provider/model database..."
    shopt -s nullglob
    local db db_filename pkg_suffix
    for db in ~/rpmbuild/SOURCES/kb-*.db; do
        db_filename="$(basename "${db}")"
        pkg_suffix="$(pkg_suffix_for "${db_filename}")"
        echo "=== Building pgedge-ai-kb-${pkg_suffix} ==="
        QA_RPATHS=$(( 0xffff )) rpmbuild -ba \
            ~/rpmbuild/SPECS/pgedge-ai-kb.spec \
            --define "pgedge_kb_version ${PGEDGE_KB_VERSION}" \
            --define "pgedge_kb_buildnum ${PGEDGE_KB_BUILDNUM}" \
            --define "pkg_suffix ${pkg_suffix}" \
            --define "db_filename ${db_filename}"
    done
}

post_build() {
    echo "Copying built RPMs to /output..."
    mkdir -p /output
    cp -v ~/rpmbuild/RPMS/*/*.rpm /output/ || echo "No binary RPMs found"
    cp -v ~/rpmbuild/SRPMS/*.src.rpm /output/ || echo "No SRPM found"

    sign_rpms /output/*.rpm
    validate_signatures /output/*.rpm
}
