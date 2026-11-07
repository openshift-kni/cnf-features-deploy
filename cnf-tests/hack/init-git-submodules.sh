#!/bin/bash

cd "$(dirname "$0")"/..

set -x
set -e

# Retry wrapper to handle transient DNS/network failures in CI
retry() {
    local max_attempts=3
    local attempt=1
    local delay=5
    while [ $attempt -le $max_attempts ]; do
        if "$@"; then
            return 0
        fi
        if [ $attempt -lt $max_attempts ]; then
            echo "'$*' failed (attempt $attempt/$max_attempts), retrying in ${delay}s..."
            sleep $delay
            delay=$((delay * 2))
        fi
        attempt=$((attempt + 1))
    done
    echo "'$*' failed after $max_attempts attempts"
    return 1
}

if [ -n "$TARGET_RELEASE" ]; then
    METALLB_OPERATOR_TARGET_COMMIT="$TARGET_RELEASE"
    SRIOV_NETWORK_OPERATOR_TARGET_COMMIT="$TARGET_RELEASE"
    CLUSTER_NODE_TUNING_OPERATOR_TARGET_COMMIT="$TARGET_RELEASE"
fi

echo "metallb-operator target commit: ${METALLB_OPERATOR_TARGET_COMMIT}"
echo "sriov-operator target commit: ${SRIOV_NETWORK_OPERATOR_TARGET_COMMIT}"
echo "cluster-node-tuning-operator target commit: ${CLUSTER_NODE_TUNING_OPERATOR_TARGET_COMMIT}"

cd submodules/metallb-operator/
retry git fetch origin
git checkout origin/"${METALLB_OPERATOR_TARGET_COMMIT}"

cd ../sriov-network-operator/
retry git fetch origin
git checkout origin/"${SRIOV_NETWORK_OPERATOR_TARGET_COMMIT}"

cd ../cluster-node-tuning-operator/
retry git fetch origin
git checkout origin/"${CLUSTER_NODE_TUNING_OPERATOR_TARGET_COMMIT}"

# cluster-node-tuning-operator's test suite need binary files to be compiled before running test
# https://github.com/openshift/cluster-node-tuning-operator/pull/1116
make vet
