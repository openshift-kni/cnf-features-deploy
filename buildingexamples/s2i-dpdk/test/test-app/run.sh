#!/bin/bash -eux

export CPU=$(cat /sys/fs/cgroup/cpuset/cpuset.cpus)
echo ${CPU}
echo ${PCIDEVICE_OPENSHIFT_IO_DPDKNIC}

if [ "$RUN_TYPE" == "testpmd" ]; then
envsubst < test-template.sh > test.sh
chmod +x test.sh
expect -f test.sh
fi

while true; do sleep inf; done;
