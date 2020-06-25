#!/bin/sh

start_build=false
last_build=$(oc get build -n dpdk -o json | jq '.items[-1].metadata.name' | tr -d '"')
if [ $last_build == "null" ]; then
    start_build=true
else
    build_status=$(oc get build -n dpdk $last_build -o json | jq '.status.phase' | tr -d '"')
    if [ $build_status == "Complete" ]; then
        exit 0
    elif [ $build_status == "Running" ]; then
        exit 1
    elif [ $build_status == "Failed" ]; then
        start_build=true
    fi
fi

if $start_build; then
    oc start-build -n dpdk s2i-dpdk
    exit 1
fi

