#!/bin/bash
set -o errexit -o nounset -o pipefail

CLEVIS=clevis
LSBLK=lsblk
DEBUG="true"
RESERVED_SLOT=31
CLEVIS_CONFIG_RESERVED_SLOT="$RESERVED_SLOT: tpm2 '{\"hash\":\"sha256\",\"key\":\"ecc\"}'"
TRUE=0
FALSE=1

#set -x

# log function. Takes 2 arguments:
# log level: debug or info
# string to print
log() {
	local logLevel logText

	logLevel="$1"
	logText="$2"
	case $logLevel in
	"debug")
		echo "DEBUG - $logText" >&2
		;;
	"info")
		echo "INFO - $logText" >&2
		;;
	*)
		# Code to execute when no patterns match
		;;
	esac
}

# logs a string with a debug level
logDebug() {
	local logText="$1"
	if ! [ -v DEBUG ] || { [ -v DEBUG ] && [ "$DEBUG" == "true" ]; }; then
		log "debug" "$logText"
	fi
}

# logs a string with a info level
logInfo() {
	local logText

	logText="$1"
	log "info" "$logText"
}

# return $TRUE id the temporary reserved slot is configured with a key (to disable PCR protection), returns $FALSE otherwise
isReservedSlotPresent() {
	local devicePath

	devicePath="$1"
	RESULT=$($CLEVIS luks list -d "$devicePath" -s $RESERVED_SLOT || true)
	if [ -n "$RESULT" ] && [ "$RESULT" == "$CLEVIS_CONFIG_RESERVED_SLOT" ]; then
		logDebug "reserved slot $RESERVED_SLOT is present"
		return $TRUE
	fi
	logDebug "reserved slot $RESERVED_SLOT is not present"
	return $FALSE
}

# create a temporary key in the reserved slot to disable PCR protection
addReservedSlot() {
	local reservedSlotPresent devicePath slot pcrIDs clevisConfig

	reservedSlotPresent="$1"
	devicePath="$2"
	slot="$3"
	pcrIDs="$4"
	clevisConfig="$5"
	logInfo "reservedSlotPresent=$reservedSlotPresent device=$devicePath slot=$slot with PCR IDs=$pcrIDs and $CLEVIS config=$clevisConfig"
	if [ "$reservedSlotPresent" == "$TRUE" ]; then
		logInfo "reserve slot already present, no need to add again"
		$CLEVIS luks list -d "$devicePath" || true
		return
	fi
	logInfo "adding reserved slot on device=$devicePath"
	ANYPASS=$(openssl rand -base64 21)
	echo -e "$ANYPASS\n" | $CLEVIS luks bind -s $RESERVED_SLOT -d "$devicePath" tpm2 '{}' || true
	$CLEVIS luks list -d "$devicePath" || true
}

# remove the temporary key in the reserved slot to enable PCR protection
removeReservedSlot() {
	local devicePath

	devicePath="$1"
	logInfo "removing luks reserved slot 31 in disk $devicePath"
	# do not change this line. There is a very weird behavior where variable 
	# substitution does not work for the clevis luks unbind command
	echo "sudo $CLEVIS luks unbind -s $RESERVED_SLOT -d $devicePath -f" | bash || true
}

#gets the list of luks devices in the system
getLUKSDevices() {
	local results
	results=$($LSBLK -o NAME,FSTYPE -l | grep crypto_LUKS | awk '{printf "/dev/" $1 "|"}')
	logDebug "got luks devices across all drives: $results"
	echo "$results"
}

# create a list of slot configuration for all encrypted devices in the system
parseClevisConfig() {
	local luksDevices IFS

	luksDevices="$1"
	IFS="|"
	for device in $luksDevices; do
		logDebug "device=$device"
		isReservedSlotPresent "$device"
		isReserved="$?"
		pcrSlots=$(getPcrSlotsForDevice "$device")
		logDebug "pcrSlots=$pcrSlots"
		parseClevisRegex "$pcrSlots" "$isReserved" "$device"
	done
}

getPcrSlotsForDevice() {
	local devicePath

	devicePath="$1"

	logDebug "getPcrSlotsForDevice, device=$devicePath"
	$CLEVIS luks list -d "$devicePath" | grep -v "$RESERVED_SLOT:" | grep pcr_ids || true
}

parseClevisRegex() {
	local clevisSlotsOutputWithPCR isReserved devicePath IFS

	clevisSlotsOutputWithPCR="$1"
	isReserved="$2"
	devicePath="$3"
	IFS=$'\n'
	for line in $clevisSlotsOutputWithPCR; do
		logDebug "line=$line"
		echo "$line" | sed -E 's@([0-9]+)(:\s+.*+\s+'\'')(\{)(.*?"pcr_ids":")([^"]*)(".*)(.*)('\''.*)@'"$isReserved"'|'"$devicePath"'|\1|\5|\3\4\5\6\7@'
	done
}

# executes a function pointer passed argument "functionToRun" for each slot configured with PCR and
# for every device in the system
processPCRentriesOnly() {
	local luksDevices parsedClevis functionToRun
	functionToRun="$1"
	luksDevices=$(getLUKSDevices)
	parsedClevis=$(parseClevisConfig "$luksDevices")

	if [ "$parsedClevis" == "" ]; then
		logInfo "no pcr config detected, nothing to do for $functionToRun"
		return
	fi
	logInfo "parsed clevis for all drives: $parsedClevis"
	echo "$parsedClevis" | while IFS= read -r line; do
		logDebug "$line"
		IFS="|" read -ra values <<<"$line"
		reservedSlotPresent=${values[0]}
		device=${values[1]}
		slotNumber=${values[2]}
		pcrIDs=${values[3]}
		clevisConfig=${values[4]}
		logInfo "reservedSlot=$reservedSlotPresent device=$device slot=$slotNumber with PCR IDs=$pcrIDs and clevis config=$clevisConfig"
		if [ -n "$pcrIDs" ]; then
			logDebug "before applying command: $(/usr/bin/tpm2_pcrread sha256:"$pcrIDs")"
			"$functionToRun" "$reservedSlotPresent" "$device" "$slotNumber" "$pcrIDs" "$clevisConfig" || true
			logDebug "after applying command: $(/usr/bin/tpm2_pcrread sha256:"$pcrIDs")"
		fi
	done
}

# initialize the array of upgrade detection methods serverUpdateDetectionMethods
initUpgradeDetectionMethods() {
	# shellcheck source=hwupgrade-detection-methods/file.sh
	for f in "$SCRIPT_DIR"/hwupgrade-detection-methods/*.sh; do source "$f"; done
	logInfo "detected system upgrade detection plugins:"
	for element in "${serverUpdateDetectionMethods[@]}"; do echo "$element"; done
}

# execute all hw upgrade detection functions in hwupgrade-detection-methods directory
# returns true if a hw upgrade is detected
# false otherwise
isSystemUpdating() {
	local isUpdating

	isUpdating=$FALSE
	# Iterate through the updated array and call each function
	for func in "${serverUpdateDetectionMethods[@]}"; do
		if $func; then
			isUpdating=$TRUE
			logInfo "detected update via $func"
		else
			logInfo "no update detected via $func"
		fi
	done
	return $isUpdating
}

#rebinds a given key slot that is configured with PCR for a given device
rebindPCRentriesOnly() {
	local reservedSlotPresent devicePath slot pcrIDs clevisConfig

	reservedSlotPresent="$1"
	devicePath="$2"
	slot="$3"
	pcrIDs="$4"
	clevisConfig="$5"

	logInfo "Rebinding reservedSlotPresent=$reservedSlotPresent device=$devicePath slot=$slot with PCR IDs=$pcrIDs and clevis config=$clevisConfig"
	clevis-luks-regen -d "$devicePath" -s "$slot" -q || true
	removeReservedSlot "$devicePath"
}
