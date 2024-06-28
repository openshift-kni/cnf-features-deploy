#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CNF_FEATURE_DEPLOY_ROOT=$SCRIPT_DIR/../../../..

# PGT
PGT_KUSTOMIZE_DIR="$CNF_FEATURE_DEPLOY_ROOT"/ztp/tools/pgt2acmpg/kustomize/ran.openshift.io/v1/policygentemplate
mkdir -p "$PGT_KUSTOMIZE_DIR"
"$SCRIPT_DIR"/build-pgt-plugin.sh "$PGT_KUSTOMIZE_DIR"
