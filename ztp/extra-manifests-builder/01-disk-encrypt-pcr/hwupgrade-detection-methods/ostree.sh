#!/bin/bash
set -o errexit -o nounset -o pipefail

isOstreeUpdating() {
	local RESULT

	RESULT=$(ostree admin status | grep -E "staged|pending")
	if [ "$RESULT" != "" ]; then
		return "$TRUE"
	else
		return "$FALSE"
	fi
}

# Add a new function to the array of update detection methods
serverUpdateDetectionMethods+=("isOstreeUpdating")
