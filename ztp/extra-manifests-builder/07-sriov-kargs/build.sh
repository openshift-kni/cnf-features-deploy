#!/bin/bash

MCPROLE=${MCPROLE:-master}

# This MachineConfig includes additional SR-IOV-related arguments in the kernel command line

echo "\
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: ${MCPROLE}
  name: 07-sriov-related-kernel-args-${MCPROLE}
spec:
  config:
    ignition:
      version: 3.2.0
  kernelArguments:
    - intel_iommu=on
    - iommu=pt"
