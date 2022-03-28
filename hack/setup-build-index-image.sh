#!/bin/bash

set -e
. $(dirname "$0")/common.sh

#Note: build the index image for all the images we need
dockercgf=`oc -n openshift-marketplace get sa builder -oyaml | grep imagePullSecrets -A 1 | grep -o "builder-.*"`

export OPERATOR_RELEASE="release-${OPERATOR_VERSION}"

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

          git clone --single-branch --branch OPERATOR_RELEASES https://github.com/openshift/sriov-network-operator.git
          cd sriov-network-operator
          podman build -f bundleci.Dockerfile --tag image-registry.openshift-image-registry.svc:5000/openshift-marketplace/sriov-operator-bundle:latest .
          podman push image-registry.openshift-image-registry.svc:5000/openshift-marketplace/sriov-operator-bundle:latest --tls-verify=false
          cd ..

          git clone --single-branch --branch OPERATOR_RELEASES https://github.com/openshift/ptp-operator.git
          cd ptp-operator
          podman build -f bundleci.Dockerfile --tag image-registry.openshift-image-registry.svc:5000/openshift-marketplace/ptp-operator-bundle:latest .
          podman push image-registry.openshift-image-registry.svc:5000/openshift-marketplace/ptp-operator-bundle:latest --tls-verify=false
          cd ..


          git clone --single-branch --branch OPERATOR_RELEASES https://github.com/openshift/special-resource-operator.git
          cd special-resource-operator/bundle/OPERATOR_VERSION/
          podman build -f bundle.Dockerfile --tag image-registry.openshift-image-registry.svc:5000/openshift-marketplace/special-resource-operator-bundle:latest .
          podman push image-registry.openshift-image-registry.svc:5000/openshift-marketplace/special-resource-operator-bundle:latest --tls-verify=false
          cd ../../..

          git clone --single-branch --branch OPERATOR_RELEASES https://github.com/openshift/cluster-nfd-operator.git
          cd cluster-nfd-operator/manifests/OPERATOR_VERSION/
          podman build -f bundle.Dockerfile --tag image-registry.openshift-image-registry.svc:5000/openshift-marketplace/cluster-nfd-operator-bundle:latest .
          podman push image-registry.openshift-image-registry.svc:5000/openshift-marketplace/cluster-nfd-operator-bundle:latest --tls-verify=false
          cd ../../..


          git clone --single-branch --branch OPERATOR_RELEASES https://github.com/openshift/metallb-operator.git
          cd metallb-operator
          podman build -f bundleci.Dockerfile --tag image-registry.openshift-image-registry.svc:5000/openshift-marketplace/metallb-operator-bundle:latest .
          podman push image-registry.openshift-image-registry.svc:5000/openshift-marketplace/metallb-operator-bundle:latest --tls-verify=false
          cd ..


          git clone --single-branch --branch GATEKEEPER_VERSION https://github.com/open-cluster-management/gatekeeper-operator.git
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

jobdefinition=$(sed "s#OPERATOR_VERSION#${OPERATOR_VERSION}#" <<< "$jobdefinition")
jobdefinition=$(sed "s#OPERATOR_RELEASES#${OPERATOR_RELEASE}#" <<< "$jobdefinition")
jobdefinition=$(sed "s#GATEKEEPER_VERSION#${GATEKEEPER_VERSION}#" <<< "$jobdefinition")


jobdefinition="${jobdefinition} secretName: ${dockercgf}"
echo "$jobdefinition"
echo "$jobdefinition" | ${OC_TOOL} apply -f -

success=0
iterations=0
sleep_time=10
max_iterations=72 # results in 12 minutes timeout
until [[ $success -eq 1 ]] || [[ $iterations -eq $max_iterations ]]
do
  run_status=$(${OC_TOOL} -n openshift-marketplace get pod podman -o json | jq '.status.phase' | tr -d '"')
   if [ "$run_status" == "Succeeded" ]; then
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

# print the build logs
${OC_TOOL} -n openshift-marketplace logs podman

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
