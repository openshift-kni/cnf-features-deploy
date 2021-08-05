#!/bin/bash
set -eu

export DRIVER_TOOLKIT_IMAGE=$1; shift
export OPENSHIFT_SECRET_FILE=$1; shift

podman pull --authfile ${OPENSHIFT_SECRET_FILE} ${DRIVER_TOOLKIT_IMAGE} >/dev/null

podman rm -f driver-toolkit >/dev/null || true
podman run -it -d --entrypoint /bin/bash --name driver-toolkit ${DRIVER_TOOLKIT_IMAGE} >/dev/null

CORE_FIND="kernel-core-"
RT_CORE_FIND="kernel-rt-core-"

KERNEL_CORE_FILE=$(podman exec driver-toolkit rpm -qa | grep ${CORE_FIND})
RT_KERNEL_CORE_FILE=$(podman exec driver-toolkit rpm -qa | grep ${RT_CORE_FIND})

export KERNEL_VERSION=`echo $KERNEL_CORE_FILE | sed "s#${CORE_FIND}##"`
export KERNEL_RT_VERSION=`echo $RT_KERNEL_CORE_FILE | sed "s#${RT_CORE_FIND}##"`

podman rm driver-toolkit -f >/dev/null

echo $KERNEL_VERSION,$KERNEL_RT_VERSION
