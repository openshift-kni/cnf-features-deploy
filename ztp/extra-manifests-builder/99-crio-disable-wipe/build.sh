#!/bin/bash

MCPROLE=${MCPROLE:-master}

crio_disable_wipe_base64=$(base64 -w0 crio-disable-wipe.toml)

echo "\
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: ${MCPROLE}
  name: 99-crio-disable-wipe-${MCPROLE}
spec:
  config:
    ignition:
      version: 3.2.0
    storage:
      files:
        - contents:
            source: data:text/plain;charset=utf-8;base64,${crio_disable_wipe_base64}
          mode: 420
          path: /etc/crio/crio.conf.d/99-crio-disable-wipe.toml"
