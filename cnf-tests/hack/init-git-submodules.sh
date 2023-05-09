#!/bin/bash

cd "$(dirname "$0")"/..

set +x
set -e

echo "metallb-operator target commit: ${METALLB_OPERATOR_TARGET_COMMIT}"
echo "sriov-operator target commit: ${SRIOV_NETWORK_OPERATOR_TARGET_COMMIT}"
echo "ptp-operator target commit: ${PTP_OPERATOR_TARGET_COMMIT}"
echo "cluster-node-tuning-operator target commit: ${CLUSTER_NODE_TUNING_OPERATOR_TARGET_COMMIT}"

cd submodules/metallb-operator/
git submodule update --init
git fetch --all
git checkout "${METALLB_OPERATOR_TARGET_COMMIT}"

cd ../sriov-network-operator/
git submodule update --init
git fetch --all
git checkout "${SRIOV_NETWORK_OPERATOR_TARGET_COMMIT}"

cd ../ptp-operator/
git submodule update --init
git fetch --all
git checkout "${PTP_OPERATOR_TARGET_COMMIT}"

cd ../cluster-node-tuning-operator/
git submodule update --init
git fetch --all
git checkout "${CLUSTER_NODE_TUNING_OPERATOR_TARGET_COMMIT}"
