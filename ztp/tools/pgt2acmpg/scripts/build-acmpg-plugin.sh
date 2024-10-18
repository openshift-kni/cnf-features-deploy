#!/bin/bash
set -o errexit -o nounset -o pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CNF_FEATURE_DEPLOY_ROOT=$SCRIPT_DIR/../../../..

# Due to https://github.com/golang/go/issues/44840 we are unable to use go install
# Instead we will download the source code and compile it ourselves
TMP_DIR=$(mktemp -d -p .)
echo "Created ${TMP_DIR}"
pushd "${TMP_DIR}"

# The details we will use to query ACM
ACM_FORK="stolostron"
ACM_BRANCH="main"

# We need to check what version of the policy generator tag is in use
POLICY_GENERATOR_TAG=$(
    curl "https://raw.githubusercontent.com/${ACM_FORK}/multicloud-operators-subscription/refs/heads/${ACM_BRANCH}/build/Dockerfile" \
    | awk '/ENV POLICY_GENERATOR_TAG=/ {print $2}' | cut -d= -f2-
)

# Print the tag to make diagnosing issues easier
echo "Detected POLICY_GENERATOR_TAG=${POLICY_GENERATOR_TAG}"

# Download the matching branch
git clone --depth 1 --branch "${POLICY_GENERATOR_TAG}" --single-branch "https://github.com/${ACM_FORK}/policy-generator-plugin.git"

# build binary and copy it out
pushd "policy-generator-plugin"
go mod vendor
make build-binary
cp PolicyGenerator "$1"

# popd twice to get back where we started
popd
popd

# cleanup directory
rm -rf "${TMP_DIR}"
echo "Deleted ${TMP_DIR}"
