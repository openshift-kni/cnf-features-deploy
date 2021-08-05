#!/bin/bash
set -eu

ACTION=$1; shift
IMAGE=$1; shift
KERNEL=`uname -r`

podman pull --authfile /var/lib/kubelet/config.json ${IMAGE}:${KERNEL} 2>&1

load_kmods() {

    podman run -i --privileged -v /lib/modules/${KERNEL}/kernel/drivers/:/lib/modules/${KERNEL}/kernel/drivers/ ${IMAGE}:${KERNEL} load.sh
}
unload_kmods() {
    podman run -i --privileged -v /lib/modules/${KERNEL}/kernel/drivers/:/lib/modules/${KERNEL}/kernel/drivers/ ${IMAGE}:${KERNEL} unload.sh
}

case "${ACTION}" in
    load)
        load_kmods
    ;;
    unload)
        unload_kmods
    ;;
    *)
        echo "Unknown command. Exiting."
        echo "Usage:"
        echo ""
        echo "load        Load kernel module(s)"
        echo "unload      Unload kernel module(s)"
        exit 1
esac
