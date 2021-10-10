#!/bin/bash

set -e
. $(dirname "$0")/common.sh

export NON_PTP_LABEL="${NON_PTP_LABEL:-node-role.kubernetes.io/virtual}"

# Label worker nodes as worker-cnf
nodes=$(${OC_TOOL} get nodes --selector='node-role.kubernetes.io/worker' \
  --selector='!node-role.kubernetes.io/master' -o name | sed -n 1,2p)
echo "[INFO]: Labeling $(echo "${nodes}" | wc -w) worker nodes with worker-cnf"
for node in $nodes
do
    ${OC_TOOL} label --overwrite $node node-role.kubernetes.io/worker-cnf=""
done

echo "[INFO]: Labeling first node as the ptp grandmaster"
node=$(${OC_TOOL} get nodes -o name --selector "!${NON_PTP_LABEL}" | sed -n 1p)
${OC_TOOL} label --overwrite $node ptp/grandmaster=""

echo "[INFO]: Labeling all the other nodes as ptp slaves"
nodes=$(${OC_TOOL} get nodes -o name --selector "!${NON_PTP_LABEL}" | sed 1d)
for node in $nodes
do
    ${OC_TOOL} label --overwrite $node ptp/slave=""
done

# Note: this is intended to be the only pool we apply all mcs to.
# Additional mcs must be added to this poll in the selector
cat <<EOF | ${OC_TOOL} apply -f -
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfigPool
metadata:
  name: worker-cnf
  labels:
    machineconfiguration.openshift.io/role: worker-cnf
spec:
  machineConfigSelector:
    matchExpressions:
      - {
          key: machineconfiguration.openshift.io/role,
          operator: In,
          values: [worker-cnf, worker],
        }
  paused: false
  nodeSelector:
    matchLabels:
      node-role.kubernetes.io/worker-cnf: ""
---
EOF

# Note: Patch the openshift network operator.
# Add a dummy dhcp network to start the dhcp daemonset by the operator.
# https://docs.openshift.com/container-platform/4.3/networking/multiple-networks/configuring-sr-iov.html#nw-multus-ipam-object_configuring-sr-iov
oc patch networks.operator.openshift.io cluster --type='merge' \
      -p='{"spec":{"additionalNetworks":[{"name":"dummy-dhcp-network","simpleMacvlanConfig":{"ipamConfig":{"type":"dhcp"},"master":"eth0","mode":"bridge","mtu":1500},"type":"SimpleMacvlan"}]}}'

#Note: build the index image for all the images we need
dockercgf=`oc -n openshift-marketplace get sa builder -oyaml | grep imagePullSecrets -A 1 | grep -o "builder-.*"`

# TODO: improve the script
jobdefinition='apiVersion: v1
kind: Pod
metadata:
  name: podman
  namespace: openshift-marketplace
spec:
  restartPolicy: Never
  serviceAccountName: builder
  containers:
    - name: priv
      image: quay.io/podman/stable
      command:
        - /bin/bash
        - -c
        - |
          set -xe

          yum install jq git wget -y
          wget https://github.com/operator-framework/operator-registry/releases/download/v1.19.0/linux-amd64-opm
          mv linux-amd64-opm opm
          chmod +x ./opm
          export pass=$( jq .\"image-registry.openshift-image-registry.svc:5000\".password /var/run/secrets/openshift.io/push/.dockercfg )
          podman login -u serviceaccount -p ${pass:1:-1} image-registry.openshift-image-registry.svc:5000 --tls-verify=false

          git clone --single-branch --branch release-4.9 https://github.com/openshift/sriov-network-operator.git
          cd sriov-network-operator
          podman build -f bundleci.Dockerfile --tag image-registry.openshift-image-registry.svc:5000/openshift-marketplace/sriov-operator-bundle:latest .
          podman push image-registry.openshift-image-registry.svc:5000/openshift-marketplace/sriov-operator-bundle:latest --tls-verify=false
          cd ..

          git clone --single-branch --branch release-4.9 https://github.com/openshift/ptp-operator.git
          cd ptp-operator
          podman build -f bundleci.Dockerfile --tag image-registry.openshift-image-registry.svc:5000/openshift-marketplace/ptp-operator-bundle:latest .
          podman push image-registry.openshift-image-registry.svc:5000/openshift-marketplace/ptp-operator-bundle:latest --tls-verify=false
          cd ..


          git clone --single-branch --branch release-4.9 https://github.com/openshift/special-resource-operator.git
          cd special-resource-operator/bundle/4.9/
          podman build -f bundle.Dockerfile --tag image-registry.openshift-image-registry.svc:5000/openshift-marketplace/special-resource-operator-bundle:latest .
          podman push image-registry.openshift-image-registry.svc:5000/openshift-marketplace/special-resource-operator-bundle:latest --tls-verify=false
          cd ../../..

          git clone --single-branch --branch release-4.9 https://github.com/openshift/cluster-nfd-operator.git
          cd cluster-nfd-operator/manifests/4.9/
          podman build -f bundle.Dockerfile --tag image-registry.openshift-image-registry.svc:5000/openshift-marketplace/cluster-nfd-operator-bundle:latest .
          podman push image-registry.openshift-image-registry.svc:5000/openshift-marketplace/cluster-nfd-operator-bundle:latest --tls-verify=false
          cd ../../..


          git clone --single-branch --branch release-4.9 https://github.com/openshift/metallb-operator.git
          cd metallb-operator
          podman build -f bundleci.Dockerfile --tag image-registry.openshift-image-registry.svc:5000/openshift-marketplace/metallb-operator-bundle:latest .
          podman push image-registry.openshift-image-registry.svc:5000/openshift-marketplace/metallb-operator-bundle:latest --tls-verify=false
          cd ..


          git clone --single-branch --branch v0.2.0 https://github.com/open-cluster-management/gatekeeper-operator.git
          cd gatekeeper-operator
          podman build -f bundle.Dockerfile --tag image-registry.openshift-image-registry.svc:5000/openshift-marketplace/gatekeeper-operator-bundle:latest .
          podman push image-registry.openshift-image-registry.svc:5000/openshift-marketplace/gatekeeper-operator-bundle:latest --tls-verify=false
          cd ..

          ./opm index --skip-tls add --bundles image-registry.openshift-image-registry.svc:5000/openshift-marketplace/sriov-operator-bundle:latest,image-registry.openshift-image-registry.svc:5000/openshift-marketplace/ptp-operator-bundle:latest,image-registry.openshift-image-registry.svc:5000/openshift-marketplace/special-resource-operator-bundle:latest,image-registry.openshift-image-registry.svc:5000/openshift-marketplace/cluster-nfd-operator-bundle:latest,image-registry.openshift-image-registry.svc:5000/openshift-marketplace/metallb-operator-bundle:latest,image-registry.openshift-image-registry.svc:5000/openshift-marketplace/gatekeeper-operator-bundle:latest --tag image-registry.openshift-image-registry.svc:5000/openshift-marketplace/ci-index:latest -p podman --mode semver
          podman push image-registry.openshift-image-registry.svc:5000/openshift-marketplace/ci-index:latest --tls-verify=false
      securityContext:
        privileged: true
      volumeMounts:
        - mountPath: /var/run/secrets/openshift.io/push
          name: dockercfg
          readOnly: true
  volumes:
    - name: dockercfg
      defaultMode: 384
      secret:
      '

jobdefinition="${jobdefinition} secretName: ${dockercgf}"
echo "$jobdefinition" | ${OC_TOOL} apply -f -

success=0
iterations=0
sleep_time=10
max_iterations=72 # results in 12 minutes timeout
until [[ $success -eq 1 ]] || [[ $iterations -eq $max_iterations ]]
do
  run_status=$(oc -n openshift-marketplace get pod podman -o json | jq '.status.phase' | tr -d '"')
   if [ $run_status == "Succeeded" ]; then
          success=1
          break
   fi
done

if [[ $success -eq 1 ]]; then
  echo "[INFO] index build succeeded"
else
  echo "[ERROR] index build failed"
  exit 1
fi

#Note: adding a CI index image
cat <<EOF | ${OC_TOOL} apply -f -
---
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: ci-index
  namespace: openshift-marketplace
spec:
  displayName: CI Index
  image: image-registry.openshift-image-registry.svc:5000/openshift-marketplace/ci-index:latest
  publisher: Red Hat
  sourceType: grpc
  updateStrategy:
    registryPoll:
      interval: 10m0s
---
EOF
