#!/bin/bash

set -e
. $(dirname "$0")/common.sh

export OPERATOR_RELEASE="release-${OPERATOR_VERSION}"

#Note: adding a CI index image
cat <<EOF | ${OC_TOOL} apply -f -
---
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: ci-index
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: quay.io/openshift-release-dev/ocp-release-nightly:iib-int-index-art-operators-${OPERATOR_VERSION}
  displayName: ART nightly
  publisher: grpc
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
