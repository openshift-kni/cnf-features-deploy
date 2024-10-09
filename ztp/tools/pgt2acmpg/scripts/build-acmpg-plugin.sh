#!/bin/bash
set -o errexit -o nounset -o pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CNF_FEATURE_DEPLOY_ROOT=$SCRIPT_DIR/../../../..

# Builds the latest version of policy-generator-plugin
# We can no longer use go install because of https://github.com/golang/go/issues/44840
# Instead we need to clone the policy-generator-plugin project and build the executable
TMP_DIR=$(mktemp -d -p .)
echo "Created $TMP_DIR"
cd "$TMP_DIR"
git clone --branch main --single-branch https://github.com/open-cluster-management-io/policy-generator-plugin.git
cd policy-generator-plugin
# This is the last commit where the policy generator was on go 1.21
git checkout eb5c12308072d4bc5a36e197e3cbb84cc23e7a82
go mod vendor
make build-binary
cp PolicyGenerator "$1"
rm -rf "$TMP_DIR"
echo "Deleted $TMP_DIR"
