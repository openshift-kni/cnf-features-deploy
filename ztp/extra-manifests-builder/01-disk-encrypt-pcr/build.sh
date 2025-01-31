#!/bin/bash
set -o errexit -o nounset -o pipefail

GOPATH=${GOPATH:-${HOME}/go}
GOBIN=${GOBIN:-${GOPATH}/bin}
MCMAKER=${MCMAKER:-${GOBIN}/mcmaker}
MCPROLE=${MCPROLE:-master}

${MCMAKER} -stdout -name 01-disk-encryption-rebind -mcp "${MCPROLE}" \
	file -source luks-helpers.sh -path /usr/local/bin/luks-helpers.sh -mode 0755 \
	file -source disablePcrOnRebootOrShutdown.sh -path /usr/local/bin/disablePcrOnRebootOrShutdown.sh -mode 0755 \
	file -source rebindDiskOnBoot.sh -path /usr/local/bin/rebindDiskOnBoot.sh -mode 0755 \
	file -source hwupgrade-detection-methods/file.sh -path /usr/local/bin/hwupgrade-detection-methods/file.sh -mode 0755 \
	file -source hwupgrade-detection-methods/fwup.sh -path /usr/local/bin/hwupgrade-detection-methods/fwup.sh -mode 0755 \
	file -source hwupgrade-detection-methods/ostree.sh -path /usr/local/bin/hwupgrade-detection-methods/ostree.sh -mode 0755 \
	file -source hwupgrade-detection-methods/talm.sh -path /usr/local/bin/hwupgrade-detection-methods/talm.sh -mode 0755 \
	file -source order.conf -path /etc/systemd/system/crio-.scope.d/order.conf -mode 0644 \
	unit -source pcr-rebind-boot.service \
	unit -source pcr-disable-shutdown.service
