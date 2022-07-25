#!/bin/bash -eux

export CPU=$(cat /sys/fs/cgroup/cpuset/cpuset.cpus)
echo ${CPU}
echo ${PCIDEVICE_OPENSHIFT_IO_DPDKNIC}

./dpdk-l2fwd -l ${CPU} -a ${PCIDEVICE_OPENSHIFT_IO_DPDKNIC} --iova-mode=va --log-level="*:debug" -- -p 1