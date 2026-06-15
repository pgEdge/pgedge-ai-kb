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

# DEB only: move a pre-release pretag (e.g. BUILDNUM='beta3_1') into the
# upstream VERSION with a leading '~' (1.0.0~beta3, BUILDNUM=1) so '~'
# sorts pre-releases BELOW stable in dpkg/reprepro.
if command -v apt-get &>/dev/null; then
    if [[ "$PGEDGE_KB_BUILDNUM" == *_* ]]; then
        PGEDGE_KB_PRETAG="${PGEDGE_KB_BUILDNUM%%_*}"
        export PGEDGE_KB_VERSION="${PGEDGE_KB_VERSION}~${PGEDGE_KB_PRETAG}"
        PGEDGE_KB_BUILDNUM="${PGEDGE_KB_BUILDNUM##*_}"
    fi
fi
