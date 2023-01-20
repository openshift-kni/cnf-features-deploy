#!/bin/bash

MCPROLE=${MCPROLE:-master}

# kdump-config MachineConfig removes the ice module from kdump to prevent kdump failures on
# certain servers. This is a temporary workaround for RHELPLAN-138236 and can be removed when
# that issue is fixed.

kdump_remove_ice_module_base64=$(base64 -w0 kdump-remove-ice-module.sh)

echo "\
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: ${MCPROLE}
  name: 05-kdump-config-${MCPROLE}
spec:
  config:
    ignition:
      version: 3.2.0
    systemd:
      units:
      - enabled: true
        name: kdump-remove-ice-module.service
        contents: |
          [Unit]
          Description=Remove ice module when doing kdump
          Before=kdump.service
          [Service]
          Type=oneshot
          RemainAfterExit=true
          ExecStart=/usr/local/bin/kdump-remove-ice-module.sh
          [Install]
          WantedBy=multi-user.target
    storage:
      files:
        - contents:
            source: data:text/plain;charset=utf-8;base64,${kdump_remove_ice_module_base64}
          mode: 448
          path: /usr/local/bin/kdump-remove-ice-module.sh"
