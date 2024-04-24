#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CNF_FEATURE_DEPLOY_ROOT=$SCRIPT_DIR/../../../..

"$SCRIPT_DIR"/build-pgt-plugin-only.sh

# Download ACM policy-generator-plugin
ACM_KUSTOMIZE_DIR="$CNF_FEATURE_DEPLOY_ROOT"/ztp/tools/pgt2acmpg/kustomize/policy.open-cluster-management.io/v1/policygenerator
mkdir -p "$ACM_KUSTOMIZE_DIR"
GOBIN="$ACM_KUSTOMIZE_DIR" go install open-cluster-management.io/policy-generator-plugin/cmd/PolicyGenerator@v1.12.4
