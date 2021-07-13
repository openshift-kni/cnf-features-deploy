#!/bin/bash

set -e

. $(dirname "$0")/common.sh

RPMS_SRC="${RPMS_SRC:-}"
REMOVE_PACKAGES="${REMOVE_PACKAGES:-}"
RPMS_NODE_ROLE="${RPMS_NODE_ROLE:-node-role.kubernetes.io/worker}"

if [ "$RPMS_SRC" == "" ]; then
	echo "[ERROR]: No RPMS_SRC provided"
	exit 1
fi

cat <<EOF | oc apply -f -
kind: ConfigMap
apiVersion: v1
metadata:
  name: rpms-entrypoint
data:
  entrypoint.sh: |
    #!/bin/sh

    set -euo pipefail
        
    LOG_FILE="/opt/log/installed_rpms.txt"
    LOG_DIR=\$(dirname \${LOG_FILE})
    TEMP_DIR=\$(mktemp -d)

    function finish {
      rm -Rf \${TEMP_DIR}
    }
    trap finish EXIT

    if [ -f "\${LOG_FILE}" ]; then
      if grep -Fxq "\${RPMS_CM_ID}" \${LOG_FILE}; then
        echo "rpms are already installed!"
        exit 0
      fi
    fi

    rpm-ostree reset

    # Fetch required packages
    install_rpms=""
    for rpm in ${RPMS_SRC}; do
      rpm_name=\$(echo \${rpm} | rev | cut -d '/' -f 1 | rev)
      install_rpms="\${install_rpms} \${rpm_name}"
      curl -s \${rpm} -o \${TEMP_DIR}/\${rpm_name}
    done
    
    if [ "${REMOVE_PACKAGES}" != "" ]; then
      rpm_install_cmd="rpm-ostree override remove ${REMOVE_PACKAGES}"
      for rpm in \${install_rpms}; do
        rpm_install_cmd="\${rpm_install_cmd} --install=\${TEMP_DIR}/\${rpm}"
      done   
      \${rpm_install_cmd}
    else
      for rpm in \${install_rpms}; do
        rpm-ostree install \${TEMP_DIR}/\${rpm}
      done
    fi

    mkdir -p \${LOG_DIR}
    rm -rf \${LOG_FILE}
    echo \${RPMS_CM_ID} > \${LOG_FILE}

    rm -Rf \${TEMP_DIR}

    # Reboot to apply changes
    systemctl reboot
EOF

cm_id=$(${OC_TOOL} get configmap rpms-entrypoint -o json | jq '.metadata.resourceVersion' | tr -d '"')

cat <<EOF | oc apply -f -
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: rpms-ds
  labels:
    app: rpms-ds
spec:
  selector:
    matchLabels:
      app: rpms-ds
  template:
    metadata:
      labels:
        app: rpms-ds
    spec:
      hostNetwork: true
      nodeSelector:
        ${RPMS_NODE_ROLE}: ""
      containers:
      - name: rpms-loader
        image: ubi8/ubi-minimal
        command: ['sh', '-c', 'cp /script/entrypoint.sh /host/tmp && chmod +x /host/tmp/entrypoint.sh && echo "Installing rpms" && chroot /host /tmp/entrypoint.sh && sleep infinity']
        securityContext:
          privileged: true
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: FallbackToLogsOnError
        env:
          - name: RPMS_CM_ID
            value: "${cm_id}"
        volumeMounts:
        - mountPath: /script
          name: rpms-script
        - mountPath: /host
          name: host
      hostNetwork: true
      restartPolicy: Always
      terminationGracePeriodSeconds: 10
      volumes:
      - configMap:
          name: rpms-entrypoint
        name: rpms-script
      - hostPath:
          path: /
          type: Directory
        name: host
EOF
