#!/bin/bash

GOPATH=${GOPATH:-${HOME}/go}
MCMAKER=${GOPATH}/bin/mcmaker
MCPROLE=${MCPROLE:-master}

# The kubelet service config is included with the container
# mount namespace because the override of the ExecStart for
# kubelet can only be done once (cannot accumulate changes
# across multiple drop-ins).
# Defaults:
#  Max Housekeeping : 15s
#  Housekeeping : 10s
#  Eviction : 10s

${MCMAKER} -name 01-kubelet-housekeeping-interval -mcp ${MCPROLE} -stdout \
        file -source extractExecStart -path /usr/local/bin/extractExecStart -mode 0755 \
        dropin -source 01-kubelet-housekeeping-interval.conf -for kubelet.service
