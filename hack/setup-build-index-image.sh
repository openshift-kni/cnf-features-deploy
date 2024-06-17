#!/bin/bash

set -e
. $(dirname "$0")/common.sh

#Note: build the index image for all the images we need
dockercgf=`oc -n openshift-marketplace get sa builder -oyaml | grep imagePullSecrets -A 1 | grep -o "builder-.*"`

export OPERATOR_RELEASE="release-${OPERATOR_VERSION}"

# remove the old job if exist
${OC_TOOL} -n openshift-marketplace delete pod podman | true
success=0
iterations=0
sleep_time=10
max_iterations=72 # results in 12 minutes timeout
until [[ $success -eq 1 ]] || [[ $iterations -eq $max_iterations ]]
do
  run_status=$(${OC_TOOL} -n openshift-marketplace get pod | grep podman | wc -l)
   if [ "$run_status" == "0" ]; then
          success=1
          break
   fi
   iterations=$((iterations+1))
   sleep $sleep_time
done

if [[ $success -eq 1 ]]; then
  echo "[INFO] index build pod removed"
else
  echo "[ERROR] failed to remove index build pod"
  exit 1
fi

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
          ARCH="amd64"
          if [[ $(uname -m) == "aarch64" ]]; then
            ARCH="arm64"
          fi
          wget -q https://github.com/operator-framework/operator-registry/releases/download/v1.23.0/linux-${ARCH}-opm
          mv linux-${ARCH}-opm opm
          chmod +x ./opm

          set +x
          pass=$( jq .\"image-registry.openshift-image-registry.svc:5000\".auth /var/run/secrets/openshift.io/push/.dockercfg )
          pass=`echo ${pass:1:-1} | base64 -d`
          podman login -u serviceaccount -p ${pass:8} image-registry.openshift-image-registry.svc:5000 --tls-verify=false
          set -x

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
          cd special-resource-operator/bundle/SRO_VERSION/
          rm manifests/image-references
          podman build -f bundle.Dockerfile --tag image-registry.openshift-image-registry.svc:5000/openshift-marketplace/special-resource-operator-bundle:latest .
          podman push image-registry.openshift-image-registry.svc:5000/openshift-marketplace/special-resource-operator-bundle:latest --tls-verify=false
          cd ../../..

          git clone --single-branch --branch OPERATOR_RELEASES https://github.com/openshift/cluster-nfd-operator.git
          cd cluster-nfd-operator/manifests/stable/
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
  terminationGracePeriodSeconds: 5
  volumes:
    - name: dockercfg
      defaultMode: 384
      secret:
      '

jobdefinition=$(sed "s#OPERATOR_VERSION#${OPERATOR_VERSION}#" <<< "$jobdefinition")
jobdefinition=$(sed "s#OPERATOR_RELEASES#${OPERATOR_RELEASE}#" <<< "$jobdefinition")
jobdefinition=$(sed "s#GATEKEEPER_VERSION#${GATEKEEPER_VERSION}#" <<< "$jobdefinition")
jobdefinition=$(sed "s#SRO_VERSION#${SRO_VERSION}#" <<< "$jobdefinition")

${OC_TOOL} label ns openshift-marketplace --overwrite pod-security.kubernetes.io/enforce=privileged

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
   iterations=$((iterations+1))
   sleep $sleep_time
done

# print the build logs
${OC_TOOL} -n openshift-marketplace logs podman

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

# re-create the container if there is a problem pulling the image.
success=0
iterations=0
sleep_time=10
max_iterations=72 # results in 12 minutes timeout
until [[ $success -eq 1 ]] || [[ $iterations -eq $max_iterations ]]
do
  run_status=$(${OC_TOOL} -n openshift-marketplace get pod | grep ci-index | awk '{print $3}')
   if [ "$run_status" == "Running" ]; then
          success=1
          break
   elif [[ "$run_status" == *"Image"*  ]]; then
       echo "pod in bad status try to recreate the image again status: $run_status"
       pod_name=$(${OC_TOOL} -n openshift-marketplace get pod | grep ci-index | awk '{print $1}')
       ${OC_TOOL} -n openshift-marketplace delete po $pod_name
   fi
   iterations=$((iterations+1))
   sleep $sleep_time
done

${OC_TOOL} label ns openshift-marketplace --overwrite pod-security.kubernetes.io/enforce=baseline

if [[ $success -eq 1 ]]; then
  echo "[INFO] index image pod running"
else
  echo "[ERROR] index image pod failed to run"
  exit 1
fi
