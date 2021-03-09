#!/bin/bash

set -e

pushd . >&2
cd "$(dirname "$0")/.."

function finish {
    popd >&2
}
trap finish EXIT

function debug {
    echo "DEBUG: $@" >&2
}

GOPATH="${GOPATH:-~/go}"
export GOFLAGS="${GOFLAGS:-"-mod=vendor"}"

export PATH=$PATH:$GOPATH/bin

export OCP_VERSION="${OCP_VERSION:-4.8}"

export TESTS_REPORTS_PATH="${TESTS_REPORTS_PATH:-/tmp/artifacts/}"

