#!/bin/bash
set -o errexit -o nounset -o pipefail

SCRIPT_DIR=$(dirname "$0")
source "$SCRIPT_DIR"/luks-helpers.sh

logInfo "Shutting down or rebooting"
initUpgradeDetectionMethods
if isSystemUpdating; then
	logInfo "System HW update detected, disabling PCR protection on all PCR protected LUKS partitions"
	processPCRentriesOnly addReservedSlot
	exit 0
fi

logInfo "No System HW update detected, continue"
