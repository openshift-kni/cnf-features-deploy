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

export OCP_VERSION="${OCP_VERSION:-4.8}"

export TESTS_REPORTS_PATH="${TESTS_REPORTS_PATH:-/logs/artifacts/}"

# expect oc to be in PATH by default
export OC_TOOL="${OC_TOOL:-oc}"

export CONTAINER_MGMT_CLI="${CONTAINER_MGMT_CLI:-podman}"
export TESTS_IN_CONTAINER="${TESTS_IN_CONTAINER:-false}"
