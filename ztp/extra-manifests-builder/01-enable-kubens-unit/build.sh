#!/bin/bash

MCPROLE=${MCPROLE:-master}

# enable kubens systemd unit to hide kubelet namespace and reduce cpu usage

echo "\
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: ${MCPROLE}
  name: 01-enable-kubens-unit-${MCPROLE}
spec:
  config:
    ignition:
      version: 3.2.0
    systemd:
      units:
      - enabled: true
        name: kubens.service"
