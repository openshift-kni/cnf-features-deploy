#!/bin/bash

MCPROLE=${MCPROLE:-master}
SYNC_ATTEMPT_TIMEOUT_SEC=${SYNC_ATTEMPT_TIMEOUT_SEC:-300}

echo "\
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: ${MCPROLE}
  name: 99-sync-time-once-${MCPROLE}
spec:
  config:
    ignition:
      version: 3.2.0
    systemd:
      units:
        - contents: |
            [Unit]
            Description=Sync time once
            After=network.service
            [Service]
            Type=oneshot
            TimeoutStartSec=${SYNC_ATTEMPT_TIMEOUT_SEC}
            ExecStart=/usr/sbin/chronyd -n -f /etc/chrony.conf -q
            RemainAfterExit=yes
            [Install]
            WantedBy=multi-user.target
          enabled: true
          name: sync-time-once.service"
