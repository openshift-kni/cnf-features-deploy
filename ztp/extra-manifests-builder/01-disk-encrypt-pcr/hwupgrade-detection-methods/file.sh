#!/bin/bash
set -o errexit -o nounset -o pipefail

isCustomFileUpdating() {
	if [ -f "/etc/host-hw-Updating.flag" ]; then
		return "$TRUE"
	else
		return "$FALSE"
	fi
}

# Add a new function to the array of update detection methods
serverUpdateDetectionMethods+=("isCustomFileUpdating")
