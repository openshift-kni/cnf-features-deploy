#!/bin/bash

set -e

pushd .
cd "$(dirname "$0")/.."

function finish {
    popd
}
trap finish EXIT

GOPATH="${GOPATH:-~/go}"
export GOFLAGS="${GOFLAGS:-"-mod=vendor"}"

export PATH=$PATH:$GOPATH/bin

export OCP_VERSION="${OCP_VERSION:-4.21}"
export OPERATOR_VERSION="${OPERATOR_VERSION:-4.21}"
export GATEKEEPER_VERSION="${GATEKEEPER_VERSION:-v0.2.0}"
export SRO_VERSION="${SRO_VERSION:-4.11}"

# the metallb-operator deployment and test namespace
export OO_INSTALL_NAMESPACE="${OO_INSTALL_NAMESPACE:-openshift-metallb-system}"
export FRRK8S_EXTERNAL_NAMESPACE="${FRRK8S_EXTERNAL_NAMESPACE:-openshift-frr-k8s}"

export TESTS_REPORTS_PATH="${TESTS_REPORTS_PATH:-/logs/artifacts/}"
export JUNIT_TO_HTML="${JUNIT_TO_HTML:-false}"

# expect oc to be in PATH by default
export OC_TOOL="${OC_TOOL:-oc}"

export CONTAINER_MGMT_CLI="${CONTAINER_MGMT_CLI:-podman}"
export TESTS_IN_CONTAINER="${TESTS_IN_CONTAINER:-false}"
export HYPERSHIFT_ENVIRONMENT="${HYPERSHIFT_ENVIRONMENT:-false}"

# Map for the tests paths
declare -A TESTS_PATHS=\
(["configsuite nto"]="cnf-tests/submodules/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/0_config"\
 ["validationsuite integration"]="cnf-tests/testsuites/validationsuite"\
 ["validationsuite metallb"]="cnf-tests/submodules/metallb-operator/test/e2e/validation"\
 ["cnftests integration"]="cnf-tests/testsuites/e2esuite"\
 ["cnftests metallb"]="cnf-tests/submodules/metallb-operator/test/e2e/functional"\
 ["cnftests sriov"]="cnf-tests/submodules/sriov-network-operator/test/conformance"\
 ["cnftests nto-performance"]="cnf-tests/submodules/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/1_performance")
export TESTS_PATHS

get_current_commit() {
  pushd "${TESTS_PATHS[$1 $2]}"
  export CURRENT_TEST="$1 $2: $(git rev-parse --short HEAD) - $(git log -1 --pretty=%s)"
  popd
}
