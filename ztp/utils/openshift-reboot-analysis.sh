#!/bin/bash

NODE_REBOOTS=$(last -x shutdown reboot |grep reboot | wc -l)
NODE_SHUTDOWNS=$(last -x shutdown reboot |grep shutdown | wc -l)
NODE_UNGRACEFUL_REBOOTS=$(($NODE_REBOOTS - $NODE_SHUTDOWNS))
SRIOV_REBOOTS=$(oc -n openshift-sriov-network-operator logs -l app=sriov-network-config-daemon -c sriov-network-config-daemon | grep "reqReboot true")
MCO_REBOOTS=$(journalctl | grep "initiating reboot: Node will reboot" | wc -l)

echo "Node reboots: $NODE_REBOOTS"
echo "Node shutdowns $NODE_SHUTDOWNS"
echo "Node ungraceful reboots $NODE_UNGRACEFUL_REBOOTS"
echo "Node reboots due to SRIOV: $SRIOV_REBOOTS"
echo "Node reboots due to MCO: $MCO_REBOOTS"
echo "Dumping rendered MachineConfig objects..."
journalctl |grep "initiating reboot: Node will reboot" | rev | cut -d " " -f1 | rev | xargs -I {} sh -c "oc get mc -oyaml {} > {}.yaml"
