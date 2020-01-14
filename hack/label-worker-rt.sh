#!/bin/bash

set -e

# expect oc to be in PATH by default
export OC_TOOL="${OC_TOOL:-oc}"

# Label 1 worker node
echo "[INFO]: Labeling 1 worker node with worker-rt"
node=$(${OC_TOOL} get nodes --selector='node-role.kubernetes.io/worker' -o name | head -1)
${OC_TOOL} label $node node-role.kubernetes.io/worker-rt=""
