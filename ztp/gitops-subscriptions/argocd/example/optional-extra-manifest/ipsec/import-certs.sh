#!/bin/bash

FILESDIR=${FILESDIR:-$(pwd)}

for role in master worker; do
  cat > "99-ipsec-${role}-import-certs.bu" <<-EOF
  variant: openshift
  version: 4.14.0
  metadata:
    name: 99-ipsec-${role}-import-certs
    labels:
      machineconfiguration.openshift.io/role: ${role}
  systemd:
    units:
    - name: ipsec-import.service
      enabled: true
      contents: |
        [Unit]
        Description=Import external certs into ipsec NSS
        Before=ipsec.service

        [Service]
        Type=oneshot
        ExecStart=/usr/local/bin/ipsec-addcert.sh
        RemainAfterExit=false
        StandardOutput=journal

        [Install]
        WantedBy=multi-user.target
  storage:
    files:
    - path: /etc/pki/certs/ca.pem
      mode: 0400
      overwrite: true
      contents:
        local: ca.pem
    - path: /etc/pki/certs/left_server.p12
      mode: 0400
      overwrite: true
      contents:
        local: left_server.p12
    - path: /usr/local/bin/ipsec-addcert.sh
      mode: 0740
      overwrite: true
      contents:
        inline: |
          #!/bin/bash -e
          echo "importing cert to NSS"
          certutil -A -n "CA" -t "CT,C,C" -d /var/lib/ipsec/nss/ -i /etc/pki/certs/ca.pem
          pk12util -W "" -i /etc/pki/certs/left_server.p12 -d /var/lib/ipsec/nss/
          certutil -M -n "left_server" -t "u,u,u" -d /var/lib/ipsec/nss/
EOF
done

for role in master worker; do
  butane 99-ipsec-${role}-import-certs.bu -o ./99-ipsec-${role}-import-certs.yaml --files-dir ${FILESDIR}
done
