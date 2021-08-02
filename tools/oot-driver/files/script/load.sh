#!/bin/bash
set -eu

if [[ "$(uname -r)" != ${KERNEL_VERSION}  ]]
then
    echo "oot driver container was compiled for kernel version ${KERNEL_VERSION} but the running version is $(uname -r)"
    exit 1
fi

mkdir -p /lib/modules/$(uname -r)/updates/drivers/net/ethernet/intel/ice/

# Link OPAE drivers
ln -s /oot-driver/*.ko "/lib/modules/$(uname -r)/updates/drivers/net/ethernet/intel/ice/"

depmod

rmmod ice || true
modprobe ice

echo "oot ice driver loaded"
