#!/bin/bash

export OC_TOOL="${OC_TOOL:-oc}"

# If not explicitly set, try development pull secret
export ACM_PULL_SECRET_FILE="${ACM_PULL_SECRET_FILE:-/root/openshift_pull.json}"

if [[ ! -f $ACM_PULL_SECRET_FILE ]]; then
  echo "[ERROR]: ACM pull secret file does not exist."
	exit 1
fi

${OC_TOOL} create namespace open-cluster-management
${OC_TOOL} project open-cluster-management
${OC_TOOL} create secret generic acm-secret -n open-cluster-management \
  --from-file=.dockerconfigjson=$ACM_PULL_SECRET_FILE \
  --type=kubernetes.io/dockerconfigjson