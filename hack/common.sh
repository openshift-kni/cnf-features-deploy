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

export OCP_VERSION="${OCP_VERSION:-4.14}"
export OPERATOR_VERSION="${OPERATOR_VERSION:-4.14}"
export GATEKEEPER_VERSION="${GATEKEEPER_VERSION:-v0.2.0}"
export SRO_VERSION="${SRO_VERSION:-4.11}"

# the metallb-operator deployment and test namespace
export OO_INSTALL_NAMESPACE="${OO_INSTALL_NAMESPACE:-openshift-metallb-system}"

export TESTS_REPORTS_PATH="${TESTS_REPORTS_PATH:-/logs/artifacts/}"
export JUNIT_TO_HTML="${JUNIT_TO_HTML:-false}"

# expect oc to be in PATH by default
export OC_TOOL="${OC_TOOL:-oc}"

export CONTAINER_MGMT_CLI="${CONTAINER_MGMT_CLI:-podman}"
export TESTS_IN_CONTAINER="${TESTS_IN_CONTAINER:-false}"
