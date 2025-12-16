#!/bin/bash
set -o errexit -o nounset -o pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CNF_FEATURE_DEPLOY_ROOT=$SCRIPT_DIR/../../../..

# Due to https://github.com/golang/go/issues/44840 we are unable to use go install
# Instead we will download the source code and compile it ourselves
TMP_DIR=$(mktemp -d -p .)
echo "Created ${TMP_DIR}"
pushd "${TMP_DIR}"

git clone https://github.com/open-cluster-management-io/policy-generator-plugin.git --branch v1.17.0 --single-branch --depth 1

# build binary and copy it out
pushd "policy-generator-plugin"

# Allow Go to download a newer toolchain if required by the upstream module
# and enable checksum database for toolchain verification
export GOTOOLCHAIN=auto
export GOSUMDB=sum.golang.org

go mod vendor
make build-binary
cp PolicyGenerator "$1"

# popd twice to get back where we started
popd
popd

# cleanup directory
rm -rf "${TMP_DIR}"
echo "Deleted ${TMP_DIR}"
