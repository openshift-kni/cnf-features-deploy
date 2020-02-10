#!/bin/bash

set -e

# expect oc to be in PATH by default
export OC_TOOL="${OC_TOOL:-oc}"

# Label 2 worker nodes as worker-cnf
echo "[INFO]: Labeling 2 worker nodes with worker-cnf"
nodes=$(${OC_TOOL} get nodes --selector='node-role.kubernetes.io/worker' \
  --selector='!node-role.kubernetes.io/master' -o name | sed -n 1,3p)
count=1;  
for node in $nodes
do
    if [ count > 2 ]; then
        ${OC_TOOL} label $node node-role.kubernetes.io/worker-cnf-no-rt=""
    else
        ${OC_TOOL} label $node node-role.kubernetes.io/worker-cnf=""
    fi
    ((count++))
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

apply_machine_config_pool() {
    # Additional mcs must be added to this poll in the selector
    cat <<EOF | ${OC_TOOL} apply -f -
    apiVersion: machineconfiguration.openshift.io/v1
    kind: MachineConfigPool
    metadata:
      name: ${CNF_LABEL}
      labels:
        machineconfiguration.openshift.io/role: ${CNF_LABEL}
    spec:
      machineConfigSelector:
        matchExpressions:
          - {
              key: machineconfiguration.openshift.io/role,
              operator: In,
              values: [${CNF_LABEL}, worker],
            }
      maxUnavailable: null
      paused: false
      nodeSelector:
        matchLabels:
          node-role.kubernetes.io/${CNF_LABEL}: ""
    ---
EOF
}

CNF_LABEL="worker-cnf"
apply_machine_config_pool
CNF_LABEL="worker-cnf-no-rt"
apply_machine_config_pool

# Note: Patch the openshift network operator.
# Add a dummy dhcp network to start the dhcp daemonset by the operator.
# https://docs.openshift.com/container-platform/4.3/networking/multiple-networks/configuring-sr-iov.html#nw-multus-ipam-object_configuring-sr-iov
oc patch networks.operator.openshift.io cluster --type='merge' \
      -p='{"spec":{"additionalNetworks":[{"name":"dummy-dhcp-network","simpleMacvlanConfig":{"ipamConfig":{"type":"dhcp"},"master":"eth0","mode":"bridge","mtu":1500},"type":"SimpleMacvlan"}]}}'
