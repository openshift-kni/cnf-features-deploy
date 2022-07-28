#!/bin/sh

last_build=$(oc get build -n dpdk -o json | jq '.items[-1].metadata.name' | tr -d '"')
if [ $last_build != "null" ]; then
    build_status=$(oc get build -n dpdk $last_build -o json | jq '.status.phase' | tr -d '"')
    if [ $build_status == "Complete" ]; then
        exit 0
    elif [ $build_status == "Failed" ] || [ $build_status == "Error" ] || [ $build_status == "Cancelled" ]; then
        oc delete build -n dpdk $last_build
        oc start-build -n dpdk s2i-dpdk
    fi
fi

exit 1
