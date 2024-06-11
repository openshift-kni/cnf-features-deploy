#!/bin/bash -eux

export CPU=$(cat /sys/fs/cgroup/cpuset/cpuset.cpus 2>/dev/null || cat /sys/fs/cgroup/cpuset.cpus 2>/dev/null)
echo ${CPU}
echo ${PCIDEVICE_OPENSHIFT_IO_DPDKNIC}

./dpdk-l2fwd -l ${CPU} -a ${PCIDEVICE_OPENSHIFT_IO_DPDKNIC} --iova-mode=va --log-level="*:debug" -- -p 1