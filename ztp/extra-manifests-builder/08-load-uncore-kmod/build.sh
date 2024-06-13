#!/bin/bash

MCPROLE=${MCPROLE:-master}

# Load intel-uncore-frequency module 

echo "\
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: ${MCPROLE}
  name: load-uncore-module-${MCPROLE}
spec:
  config:
    ignition:
      version: 3.2.0
    storage:
      files:
        - contents:
            source: data:text/plain;charset=utf-8,intel-uncore-frequency
          filesystem: root
          mode: 420
          path: /etc/modules-load.d/intel-uncore-frequency-load.conf"
