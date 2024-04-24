#!/bin/bash
set -o errexit -o nounset -o pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CNF_FEATURE_DEPLOY_ROOT=$SCRIPT_DIR/../../../..

# Build latest Policy Generator Template plugin executable
cd "$CNF_FEATURE_DEPLOY_ROOT"/ztp/policygenerator 
make build
mkdir -p "$CNF_FEATURE_DEPLOY_ROOT"/ztp/tools/pgt2acmpg/kustomize/ran.openshift.io/v1/policygentemplate
cp policygenerator "$CNF_FEATURE_DEPLOY_ROOT"/ztp/tools/pgt2acmpg/kustomize/ran.openshift.io/v1/policygentemplate/PolicyGenTemplate

# cleanup
chmod -R 755 "$CNF_FEATURE_DEPLOY_ROOT"/ztp/tools/pgt2acmpg/build || true
rm -rf "$CNF_FEATURE_DEPLOY_ROOT"/ztp/tools/pgt2acmpg/build
