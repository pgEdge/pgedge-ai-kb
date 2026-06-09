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
CWD="$(pwd)"

export DEBIAN_FRONTEND=noninteractive

# The per-provider databases are staged into the container by the
# workflow before pgedge-builder-action runs. The host's
# pkg/kb.db.cached/ maps to /build/pkg/kb.db.cached/ in the container.
#
# This package doesn't ship the kb-builder binary or example config —
# those live elsewhere (GoReleaser tarballs and the pgedge-postgres-mcp
# server package, respectively).
KB_DB_CACHE_DIR="${KB_DB_CACHE_DIR:-/build/pkg/kb.db.cached}"

# pkg_suffix_for prints the package suffix for a database filename, e.g.
# kb-openai-text-embedding-3-small.db -> openai-text-embedding-3-small.
pkg_suffix_for() {
    local base; base="$(basename "$1")"
    base="${base#kb-}"
    printf '%s' "${base%.db}"
}

prepare() {
    setup_apt_build_env
    apt-get install -y jq sqlite3

    shopt -s nullglob
    local dbs=("${KB_DB_CACHE_DIR}"/kb-*.db)
    if [ "${#dbs[@]}" -eq 0 ]; then
        echo "::error::no kb-*.db found under ${KB_DB_CACHE_DIR}"
        echo "validate-kb-db is expected to stage them before this cell runs."
        exit 1
    fi

    # Read the kb release tag from the artifact's sidecar file. See
    # build-rpm.sh for rationale.
    local release_tag_file="${KB_DB_CACHE_DIR}/RELEASE_TAG"
    if [ -f "${release_tag_file}" ]; then
        KB_DB_RELEASE_TAG="$(cat "${release_tag_file}")"
        export KB_DB_RELEASE_TAG
        echo "kb release tag: ${KB_DB_RELEASE_TAG}"
    else
        echo "::warning::RELEASE_TAG file missing alongside kb-*.db"
    fi

    # Clear the entire build workspace. dpkg-buildpackage drops .deb files
    # in the parent of each source tree; reset so post_build() only copies
    # fresh artifacts. In CI each container is fresh, but local reruns hit
    # this.
    echo "Resetting build workspace at ${BUILD_DIR}..."
    rm -rf "${BUILD_DIR}"
    mkdir -p "${BUILD_DIR}"

    echo "Refreshing apt lists for build-dep..."
    sudo apt-get update
}

build() {
    shopt -s nullglob
    local distro; distro="$(lsb_release -cs)"
    local db db_filename pkg_suffix src
    for db in "${KB_DB_CACHE_DIR}"/kb-*.db; do
        db_filename="$(basename "${db}")"
        pkg_suffix="$(pkg_suffix_for "${db_filename}")"
        src="${BUILD_DIR}/${pkg_suffix}"
        echo "=== Building pgedge-ai-kb-${pkg_suffix} (${distro}) ==="

        rm -rf "${src}"
        mkdir -p "${src}"
        cp "${db}" "${src}/${db_filename}"
        cp "${CWD}/LICENSE.md" "${CWD}/README.md" "${src}/"
        cp -r "${CWD}/${COMPONENT_NAME}/deb/debian" "${src}/"

        # Expand the templated package name and database filename in the
        # debian/ tree (GNU sed in the build container).
        find "${src}/debian" -type f -exec sed -i \
            -e "s|@PKG_SUFFIX@|${pkg_suffix}|g" \
            -e "s|@DB_FILENAME@|${db_filename}|g" {} +

        cat > "${src}/debian/changelog" <<EOF
pgedge-ai-kb-${pkg_suffix} (${PGEDGE_KB_VERSION}-${PGEDGE_KB_BUILDNUM}.${distro}) ${distro}; urgency=medium

  * Automated build from ${GITHUB_REF_NAME:-local}
  * ${db_filename} from ${KB_DB_RELEASE_TAG:-unknown}

 -- pgEdge Team <support@pgedge.com>  $(date -R)
EOF

        ( cd "${src}" && sudo apt-get build-dep -y . && dpkg-buildpackage -us -uc -b )
    done
}

post_build() {
    echo "Copying .deb packages to /output..."
    sudo mkdir -p /output
    rename_ddeb_packages "${BUILD_DIR}"
    find "${BUILD_DIR}" -maxdepth 2 -name '*.deb' -exec sudo cp {} /output/ \; \
        || echo "No .deb packages found"
}
