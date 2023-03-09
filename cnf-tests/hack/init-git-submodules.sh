#!/bin/bash

set +x
set -e

metallb_operator_target_commit="${METALLB_OPERATOR_TARGET_COMMIT:-main}"
sriov_network_operator_target_commit="${SRIOV_NETWORK_OPERATOR_TARGET_COMMIT:-master}"
ptp_operator_target_commit="${PTP_OPERATOR_TARGET_COMMIT:-master}"
cluster_node_tuning_operator_target_commit="${CLUSTER_NODE_TUNING_OPERATOR_TARGET_COMMIT:-master}"

echo "init git submodules"
echo "metallb-operator target commit: ${metallb_operator_target_commit}"
echo "sriov-operator target commit: ${sriov_network_operator_target_commit}"
echo "ptp-operator target commit: ${ptp_operator_target_commit}"
echo "clusternode-tuning-operator target commit: ${cluster_node_tuning_operator_target_commit}"

cd cnf-tests/submodules/metallb-operator/
git submodule update --init
git checkout "${metallb_operator_target_commit}"

cd ../sriov-network-operator/
git submodule update --init
git checkout "${sriov_network_operator_target_commit}"

cd ../ptp-operator/
git submodule update --init
git checkout "${ptp_operator_target_commit}"

cd ../cluster-node-tuning-operator/
git submodule update --init
git checkout "${cluster_node_tuning_operator_target_commit}"
