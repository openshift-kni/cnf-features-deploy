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

echo "[INFO]: Pausing"
# TODO patching to prevent https://bugzilla.redhat.com/show_bug.cgi?id=1792749 from happening
# remove this once the bug is fixed
mcps=$(${OC_TOOL} get mcp --no-headers -o custom-columns=":metadata.name")
for mcp in $mcps
do
    ${OC_TOOL} patch mcp "${mcp}" -p '{"spec":{"paused":true}}' --type=merge
done
