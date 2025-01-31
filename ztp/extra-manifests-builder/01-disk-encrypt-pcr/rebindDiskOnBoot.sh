#!/bin/bash
set -o errexit -o nounset -o pipefail

SCRIPT_DIR=$(dirname "$0")
# shellcheck source=luks-helpers.sh
source "$SCRIPT_DIR"/luks-helpers.sh
#set -x

logInfo "booting... checking if rebinding disk needed"
processPCRentriesOnly rebindPCRentriesOnly
