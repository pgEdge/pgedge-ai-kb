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
export KB_DB_RELEASE_TAG="${KB_DB_RELEASE_TAG:-kb-2026-05-19}"

# -----------------------------------------------------------------------------
# Repo identity (used in download URLs)
# -----------------------------------------------------------------------------
export REPO_OWNER="${REPO_OWNER:-pgEdge}"
export REPO_NAME="${REPO_NAME:-pgedge-ai-kb}"

# -----------------------------------------------------------------------------
# Required inputs from pgedge-builder-action (or set manually for local runs)
# -----------------------------------------------------------------------------
# COMPONENT_VERSION  numeric tag base, e.g. 1.0.0 (no '-')
# COMPONENT_BUILDNUM "1" for GA, or "0.<pre>" for pre-release, e.g. 0.rc1
# REPO_TYPE          daily | staging
#
# Defaults are only for local "source pkg/common.sh && echo $..." testing.
export COMPONENT_VERSION="${COMPONENT_VERSION:-0.0.0}"
export COMPONENT_BUILDNUM="${COMPONENT_BUILDNUM:-1}"
export REPO_TYPE="${REPO_TYPE:-daily}"

# -----------------------------------------------------------------------------
# All derived values below — single source of truth, no workflow override
# -----------------------------------------------------------------------------

# Spec-facing RPM version + release.
export PGEDGE_KB_VERSION="${COMPONENT_VERSION}"   # must be free of '-'
export PGEDGE_KB_RELEASE="${COMPONENT_BUILDNUM}"  # 1 | 0.<pre> | 0.test.<ts>

# Translate buildnum → both DEB version and goreleaser-style BINARY_VERSION.
# These two reconstructions are intentionally coupled: whatever string the
# release.yml workflow tagged with (v<X>-<pre>) must be reproducible here
# from the (version, buildnum) pair alone.
#
#   buildnum=1            -> DEB 1.0.0-1            BINARY 1.0.0
#   buildnum=0.rc1        -> DEB 1.0.0~rc1-1        BINARY 1.0.0-rc1
#   buildnum=0.beta1      -> DEB 1.0.0~beta1-1      BINARY 1.0.0-beta1
#   buildnum=0.test1      -> DEB 1.0.0~test1-1      BINARY 1.0.0-test1
#
# The '~' makes dpkg sort the pre-release below the GA version; the leading
# "0." in the RPM Release does the same for rpm.
if [[ "${COMPONENT_BUILDNUM}" =~ ^0\.(.+)$ ]]; then
    _pre_tag="${BASH_REMATCH[1]}"
    export PGEDGE_KB_VERSION_DEB="${COMPONENT_VERSION}~${_pre_tag}-1"
    export BINARY_VERSION="${COMPONENT_VERSION}-${_pre_tag}"
    unset _pre_tag
else
    export PGEDGE_KB_VERSION_DEB="${COMPONENT_VERSION}-${COMPONENT_BUILDNUM}"
    export BINARY_VERSION="${COMPONENT_VERSION}"
fi

# KB_DB_VERSION = KB_DB_RELEASE_TAG with the leading "kb-" stripped.
export KB_DB_VERSION="${KB_DB_RELEASE_TAG#kb-}"

# -----------------------------------------------------------------------------
# Workflow surface
# -----------------------------------------------------------------------------
# Not consumed by pgedge-builder-action — but set so future per-pg-version
# fan-out in pgedge-enterprise-unified-packages knows this component is
# version-agnostic if we ever migrate there.
export PER_PG_VERSION="${PER_PG_VERSION:-false}"
