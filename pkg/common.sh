#!/usr/bin/env bash
#
# pgedge-postgres-mcp-kb packaging environment.
#
# Sourced by common/build.sh from pgedge-builder-action before
# common-functions.sh. Sets the version variables consumed by
# build-rpm.sh, build-deb.sh, and the RPM spec / debian rules.
#
# kb.db release tag pinned here is the single source of truth for which
# kb.db release this repo bundles. Bump it and push a v* or test-* tag
# to ship a new kb.db.

# -----------------------------------------------------------------------------
# Pinned kb.db release
# -----------------------------------------------------------------------------
# This must match a tag of a GitHub release at
# https://github.com/pgEdge/pgedge-ai-kb/releases that has bin/kb.db
# attached. The validate-kb-db job downloads kb.db from this release.
export KB_DB_RELEASE_TAG="${KB_DB_RELEASE_TAG:-kb-2026-05-22}"

# Defaults are only for local "source pkg/common.sh && echo $..." testing.
export PGEDGE_KB_VERSION="${COMPONENT_VERSION:-1.0.0}"
export PGEDGE_KB_BUILDNUM="${COMPONENT_BUILDNUM:-1}"
export REPO_TYPE="${REPO_TYPE:-daily}"

# Convert underscore to tilde for Debian/Ubuntu packaging
if command -v apt-get &>/dev/null; then
    PGEDGE_KB_BUILDNUM="$(printf '%s' "${PGEDGE_KB_BUILDNUM}" | tr '_' '~')"
fi
