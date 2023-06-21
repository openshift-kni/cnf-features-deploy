#!/bin/bash

set -e
. $(dirname "$0")/common.sh

export NON_PTP_LABEL="${NON_PTP_LABEL:-node-role.kubernetes.io/virtual}"
export ROLE_WORKER_CNF="${ROLE_WORKER_CNF:-worker-cnf}"
# Usage: CNF_NODES="node/{node1_name} node/{node2_name}"
export CNF_NODES="${CNF_NODES:-}"

if [ -n "$CNF_NODES" ]; then
  for node in $CNF_NODES
  do
      is_worker=$(${OC_TOOL} get $node -o json | jq '.metadata.labels' | grep "\"node-role.kubernetes.io/worker\"" || true)
      if [ "${is_worker}" == "" ]; then
        echo "[ERROR]: Cannot use non worker $node"
        exit 1
      fi
  done
else
  CNF_NODES=$(${OC_TOOL} get nodes --selector='node-role.kubernetes.io/worker' \
  --selector='!node-role.kubernetes.io/master' -o name | sed -n 1,2p)
  if [ -z "$CNF_NODES" ]; then
    echo "[ERROR]: Cannot label any node with [${ROLE_WORKER_CNF}]"
    exit 1
  fi
fi

# Label worker nodes
echo "[INFO]: Labeling $(echo "${CNF_NODES}" | wc -w) worker nodes with ${ROLE_WORKER_CNF}"
for node in $CNF_NODES
do
    ${OC_TOOL} label --overwrite $node node-role.kubernetes.io/${ROLE_WORKER_CNF}=""
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
  name: ${ROLE_WORKER_CNF}
  labels:
    machineconfiguration.openshift.io/role: ${ROLE_WORKER_CNF}
    pools.operator.machineconfiguration.openshift.io/${ROLE_WORKER_CNF}: ""
spec:
  machineConfigSelector:
    matchExpressions:
      - {
          key: machineconfiguration.openshift.io/role,
          operator: In,
          values: [${ROLE_WORKER_CNF}, worker],
        }
  paused: false
  nodeSelector:
    matchLabels:
      node-role.kubernetes.io/${ROLE_WORKER_CNF}: ""
---
EOF

# Note: Patch the openshift network operator.
# Add a dummy dhcp network to start the dhcp daemonset by the operator.
# https://docs.openshift.com/container-platform/4.3/networking/multiple-networks/configuring-sr-iov.html#nw-multus-ipam-object_configuring-sr-iov
oc patch networks.operator.openshift.io cluster --type='merge' \
      -p='{"spec":{"additionalNetworks":[{"name":"dummy-dhcp-network","simpleMacvlanConfig":{"ipamConfig":{"type":"dhcp"},"master":"eth0","mode":"bridge","mtu":1500},"type":"SimpleMacvlan"}]}}'
