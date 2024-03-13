#!/bin/bash
set -o errexit -o nounset -o pipefail

SCRIPT_DIR=$(dirname "$0")
source "$SCRIPT_DIR"/luks-helpers.sh
DEBUG="false"

# *** TEST isReservedSlotPresent ***"
clevisTestReservedSlot() {
	echo "$RESERVED_SLOT: tpm2 '{\"hash\":\"sha256\",\"key\":\"ecc\"}'"
}

testIsReservedSlotPresent() {
	local EXPECTED results
	echo "*** TEST isReservedSlotPresent ***"
	CLEVIS=clevisTestReservedSlot
	EXPECTED=0

	isReservedSlotPresent "/dev/sda"
	results=$?
	if [ "$results" = "$EXPECTED" ]; then
		echo "PASS"
		return "$TRUE"
	fi
	echo "FAILED"
	echo "$results"
	return "$FALSE"
}

# *** TEST getLUKSDevices ***"
lsblkSlF() {
	# lsblk -o NAME,FSTYPE -l
	echo "nvme0n1                                   
nvme0n1p1                                 vfat
nvme0n1p2                                 ext4
nvme0n1p3                                 crypto_LUKS
"
}

testGetLUKSDevices() {
	local LSBLK EXPECTED results
	LSBLK=lsblkSlF

	echo "*** TEST getLUKSDevices ***"
	EXPECTED="/dev/nvme0n1p3|"

	results=$(getLUKSDevices)
	if [ "$results" = "$EXPECTED" ]; then
		echo "PASS"
		return "$TRUE"
	fi
	echo "FAILED"
	echo "$results"
	return "$FALSE"
}

# *** TEST getPcrSlotsForDevice ***
clevisTestPcrSlotsForDevice() {
	local out

	read -r -d '' out <<EOM || true
1: tpm2 '{"hash":"sha256","key":"ecc","pcr_bank":"sha256","pcr_ids":"1,7"}'
2: tpm2 '{"hash":"sha256","key":"ecc","pcr_bank":"sha256"}'
3: sss '{"t":2,"pins":{"tpm2":[{"hash":"sha256","key":"ecc","pcr_bank":"sha256","pcr_ids":"1,7,8"}],"sss":{"t":1,"pins":{"tang":[{"url":"http://192.168.28.12:8081"}]}}}}'
EOM
	echo "$out"
}

testGetPcrSlotsForDevice() {
	local CLEVIS EXPECTED results
	CLEVIS=clevisTestPcrSlotsForDevice

	echo "*** TEST getPcrSlotsForDevice ***"
	read -r -d '' EXPECTED <<EOM || true
1: tpm2 '{"hash":"sha256","key":"ecc","pcr_bank":"sha256","pcr_ids":"1,7"}'
3: sss '{"t":2,"pins":{"tpm2":[{"hash":"sha256","key":"ecc","pcr_bank":"sha256","pcr_ids":"1,7,8"}],"sss":{"t":1,"pins":{"tang":[{"url":"http://192.168.28.12:8081"}]}}}}'
EOM
	results=$(getPcrSlotsForDevice "/dev/sda")
	if [ "$results" = "$EXPECTED" ]; then
		echo "PASS"
		return "$TRUE"
	fi
	echo "FAILED"
	echo "$results"
	return "$FALSE"
}

# *** TEST parseClevisConfig ***
clevisParseClevisConfig() {
	local out

	read -r -d '' out <<EOM || true
1: tpm2 '{"hash":"sha256","key":"ecc","pcr_bank":"sha256","pcr_ids":"1,7"}'
2: tpm2 '{"hash":"sha256","key":"ecc","pcr_bank":"sha256","pcr_ids":"1,7"}'
3: sss '{"t":2,"pins":{"tpm2":[{"hash":"sha256","key":"ecc","pcr_bank":"sha256","pcr_ids":"1,7,8"}],"sss":{"t":1,"pins":{"tang":[{"url":"http://192.168.28.12:8081"}]}}}}'
EOM
	echo "$out"
}

testParseClevisConfig() {
	local CLEVIS EXPECTED results
	CLEVIS=clevisParseClevisConfig

	echo "*** TEST parseClevisConfig ***"
	read -r -d '' EXPECTED <<EOM || true
1|/dev/sda4|1|1,7|{"hash":"sha256","key":"ecc","pcr_bank":"sha256","pcr_ids":"1,7"}
1|/dev/sda4|2|1,7|{"hash":"sha256","key":"ecc","pcr_bank":"sha256","pcr_ids":"1,7"}
1|/dev/sda4|3|1,7,8|{"t":2,"pins":{"tpm2":[{"hash":"sha256","key":"ecc","pcr_bank":"sha256","pcr_ids":"1,7,8"}],"sss":{"t":1,"pins":{"tang":[{"url":"http://192.168.28.12:8081"}]}}}}
EOM
	results=$(parseClevisConfig "/dev/sda4")
	if [ "$results" = "$EXPECTED" ]; then
		echo "PASS"
		return "$TRUE"
	fi
	echo "FAILED"
	echo "$results"
	return "$FALSE"
}

testIsReservedSlotPresent
testGetLUKSDevices
testGetPcrSlotsForDevice
testParseClevisConfig
