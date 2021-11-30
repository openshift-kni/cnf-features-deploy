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

${MCMAKER} -name container-mount-namespace-and-kubelet-conf -mcp ${MCPROLE} -stdout \
        file -source extractExecStart -path /usr/local/bin/extractExecStart -mode 0755 \
        file -source nsenterCmns -path /usr/local/bin/nsenterCmns -mode 0755 \
        unit -source container-mount-namespace.service \
        dropin -source 20-container-mount-namespace.conf -for crio.service \
        dropin -source 20-container-mount-namespace-kubelet.conf -name 20-container-mount-namespace.conf -for kubelet.service \
        dropin -source 30-kubelet-interval-tuning.conf -for kubelet.service
