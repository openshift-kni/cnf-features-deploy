#!/bin/bash

cd "$(dirname "$0")"/..

set -x
set -e

if [ -n "$TARGET_RELEASE" ]; then
    METALLB_OPERATOR_TARGET_COMMIT="$TARGET_RELEASE"
    SRIOV_NETWORK_OPERATOR_TARGET_COMMIT="$TARGET_RELEASE"
    CLUSTER_NODE_TUNING_OPERATOR_TARGET_COMMIT="$TARGET_RELEASE"
fi

SRIOV_NETWORK_OPERATOR_TARGET_COMMIT="test_1"

echo "metallb-operator target commit: ${METALLB_OPERATOR_TARGET_COMMIT}"
echo "sriov-operator target commit: ${SRIOV_NETWORK_OPERATOR_TARGET_COMMIT}"
echo "cluster-node-tuning-operator target commit: ${CLUSTER_NODE_TUNING_OPERATOR_TARGET_COMMIT}"

cd submodules/metallb-operator/
git fetch --all
git checkout origin/"${METALLB_OPERATOR_TARGET_COMMIT}"

cd ../sriov-network-operator/
git fetch --all
git checkout origin/"${SRIOV_NETWORK_OPERATOR_TARGET_COMMIT}"

cd ../cluster-node-tuning-operator/
git fetch --all
git checkout origin/"${CLUSTER_NODE_TUNING_OPERATOR_TARGET_COMMIT}"

# cluster-node-tuning-operator's test suite need binary files to be compiled before running test
# https://github.com/openshift/cluster-node-tuning-operator/pull/1116
make vet
