#!/usr/bin/env bash
#
# pgedge-ai-kb packaging environment.
#
# Sourced by common/build.sh from pgedge-builder-action before
# common-functions.sh. Sets the version variables consumed by
# build-rpm.sh, build-deb.sh, and the RPM spec / debian rules.
#
# KB_DB_RELEASE_TAG is no longer pinned here. release.yml either
# generates a fresh kb.db inline (default) or discovers the latest
# kb-* GitHub release (opt-in). The build scripts read the tag from
# the workspace file pkg/kb.db.cached/RELEASE_TAG that the workflow
# stages alongside kb.db.

# Defaults are only for local "source pkg/common.sh && echo $..." testing.
export PGEDGE_KB_VERSION="${COMPONENT_VERSION:-1.0.0}"
export PGEDGE_KB_BUILDNUM="${COMPONENT_BUILDNUM:-1}"
export REPO_TYPE="${REPO_TYPE:-daily}"

# Convert underscore to tilde for Debian/Ubuntu packaging
if command -v apt-get &>/dev/null; then
    PGEDGE_KB_BUILDNUM="$(printf '%s' "${PGEDGE_KB_BUILDNUM}" | tr '_' '~')"
fi
