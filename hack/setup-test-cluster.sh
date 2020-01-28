#!/bin/bash

set -e

# expect oc to be in PATH by default
export OC_TOOL="${OC_TOOL:-oc}"

# Label 2 worker nodes as worker-cnf
echo "[INFO]: Labeling 2 worker nodes with worker-cnf"
nodes=$(${OC_TOOL} get nodes --selector='node-role.kubernetes.io/worker' -o name | sed -n 1,2p)
for node in $nodes
do
    ${OC_TOOL} label $node node-role.kubernetes.io/worker-cnf=""
done

echo "[INFO]: Labeling first node as the ptp grandmaster"
node=$(${OC_TOOL} get nodes -o name | sed -n 1p)
${OC_TOOL} label $node ptp/grandmaster=""

echo "[INFO]: Labeling all the other nodes as ptp slaves"
nodes=$(${OC_TOOL} get nodes -o name | sed 1d)
for node in $nodes
do
    ${OC_TOOL} label $node ptp/slave=""
done

# Note: this is intended to be the only pool we apply all mcs to.
# Additional mcs must be added to this poll in the selector
cat <<EOF | ${OC_TOOL} apply -f -
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfigPool
metadata:
  name: worker-cnf
  labels:
    machineconfiguration.openshift.io/role: worker-cnf
spec:
  machineConfigSelector:
    matchExpressions:
      - {
          key: machineconfiguration.openshift.io/role,
          operator: In,
          values: [worker-cnf, worker],
        }
  maxUnavailable: null
  paused: true
  nodeSelector:
    matchLabels:
      node-role.kubernetes.io/worker-cnf: ""
---
EOF

