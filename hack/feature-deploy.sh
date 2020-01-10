#!/bin/bash

set -e

if [ "$FEATURES_ENVIRONMENT" == "" ]; then
	echo "[ERROR]: No FEATURES_ENVIRONMENT provided"
	exit 1
fi

if [ "$FEATURES" == "" ]; then
	echo "[ERROR]: No FEATURES provided"
	exit 1
fi

# expect oc to be in PATH by default
export OC_TOOL="${OC_TOOL:-oc}"

# expect kustomize to be in PATH by default
KUSTOMIZE="${KUSTOMIZE:-kustomize}"

# Label 1 worker node
echo "[INFO]:labeling 1 worker node with worker-rt"
node=$(${OC_TOOL} get nodes --selector='node-role.kubernetes.io/worker' -o name | head -1)
${OC_TOOL} label --overwrite=true $node node-role.kubernetes.io/worker-rt=""

# Override the image name when this is invoked from openshift ci
# Not ideal, but kustomize does not support env vars directly :/
if [ -n "${OPENSHIFT_BUILD_NAMESPACE}" ]; then
        echo "[INFO]: Openshift CI detected, deploying using image $FULL_REGISTRY_IMAGE"
        FULL_REGISTRY_IMAGE="registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:performance-addon-operators-registry"
        cp feature-configs/e2e-gcp/performance-operator/operator_catalogsource.patch.yaml.in feature-configs/e2e-gcp/performance-operator/operator_catalogsource.patch.yaml
        echo "  $FULL_REGISTRY_IMAGE" >> feature-configs/e2e-gcp/performance-operator/operator_catalogsource.patch.yaml
fi

# Deploy features
for feature in $FEATURES; do

	echo "[INFO]: Deploying feature '$feature' for environment '$FEATURES_ENVIRONMENT'"
	${KUSTOMIZE} build feature-configs/${FEATURES_ENVIRONMENT}/${feature}/ | ${OC_TOOL} apply -f -

  # Wait for feature
  if [ -f "feature-configs/${FEATURES_ENVIRONMENT}/${feature}/wait_for_it.sh" ]; then
    echo "[INFO]: waiting for $feature to be deployed"
    feature-configs/${FEATURES_ENVIRONMENT}/${feature}/wait_for_it.sh
    echo "[INFO]: $feature was deployed"
  fi;

done
