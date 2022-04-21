#!/bin/bash

set -e
. $(dirname "$0")/common.sh

export NON_PTP_LABEL="${NON_PTP_LABEL:-node-role.kubernetes.io/virtual}"

# Label worker nodes as worker-cnf
nodes=$(${OC_TOOL} get nodes --selector='node-role.kubernetes.io/worker' \
  --selector='!node-role.kubernetes.io/master' -o name | sed -n 1,2p)

if [ -z "$nodes" ]; then
  echo "[ERROR]: Cannot label any node with [worker-cnf]"
  exit 1
fi

echo "[INFO]: Labeling $(echo "${nodes}" | wc -w) worker nodes with worker-cnf"
for node in $nodes
do
    ${OC_TOOL} label --overwrite $node node-role.kubernetes.io/worker-cnf=""
done

echo "[INFO]: Labeling first node as the ptp grandmaster"
node=$(${OC_TOOL} get nodes -o name --selector "!${NON_PTP_LABEL}" | sed -n 1p)
${OC_TOOL} label --overwrite $node ptp/grandmaster=""

echo "[INFO]: Labeling all the other nodes as ptp slaves"
nodes=$(${OC_TOOL} get nodes -o name --selector "!${NON_PTP_LABEL}" | sed 1d)
for node in $nodes
do
    ${OC_TOOL} label --overwrite $node ptp/slave=""
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
  paused: false
  nodeSelector:
    matchLabels:
      node-role.kubernetes.io/worker-cnf: ""
---
EOF

# Note: Patch the openshift network operator.
# Add a dummy dhcp network to start the dhcp daemonset by the operator.
# https://docs.openshift.com/container-platform/4.3/networking/multiple-networks/configuring-sr-iov.html#nw-multus-ipam-object_configuring-sr-iov
oc patch networks.operator.openshift.io cluster --type='merge' \
      -p='{"spec":{"additionalNetworks":[{"name":"dummy-dhcp-network","simpleMacvlanConfig":{"ipamConfig":{"type":"dhcp"},"master":"eth0","mode":"bridge","mtu":1500},"type":"SimpleMacvlan"}]}}'
