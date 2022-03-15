#!/bin/bash

MCPROLE=${MCPROLE:-master}

# kdump MachineConfig is to enable the kdump.service and add it in the kernel command line

echo "\
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: ${MCPROLE}
  name: 06-kdump-enable-${MCPROLE}
spec:
  config:
    ignition:
      version: 3.2.0
    systemd:
      units:
      - enabled: true
        name: kdump.service
  kernelArguments:
    - crashkernel=256M"
