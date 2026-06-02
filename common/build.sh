#!/usr/bin/env bash
set -euo pipefail

COMPONENT_NAME=$1
source "$(dirname "$0")/../${COMPONENT_NAME}/common.sh"

# common-functions.sh will be copied into workspace by workflow from a private repo
COMMON_FILE="$(dirname "$0")/common-functions.sh"

if [ -f "$COMMON_FILE" ]; then
  source "$COMMON_FILE"
else
  echo "Error: $COMMON_FILE not found!" >&2
  exit 1
fi

###########
# Main
###########
detect_os_type
prepare
build
post_build
