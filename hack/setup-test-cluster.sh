#!/bin/bash

set -e

# expect oc to be in PATH by default
export OC_TOOL="${OC_TOOL:-oc}"

# Label 1 worker node as worker-rt
echo "[INFO]: Labeling 1 worker node with worker-rt"
node=$(${OC_TOOL} get nodes --selector='node-role.kubernetes.io/worker' -o name | head -1)
${OC_TOOL} label $node node-role.kubernetes.io/worker-rt=""

# Label 2 worker node as worker-sctp
echo "[INFO]: Labeling 2 worker node with worker-sctp"
nodes=$(${OC_TOOL} get nodes --selector='node-role.kubernetes.io/worker' -o name | sed -n 2,3p)
for node in $nodes
do
    ${OC_TOOL} label $node node-role.kubernetes.io/worker-sctp=""
done
