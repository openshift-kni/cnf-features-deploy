#!/bin/bash

set -e
set +x

. $(dirname "$0")/common.sh

if ! which go; then
  echo "No go command available"
  exit 1
fi

GOPATH="${GOPATH:-~/go}"
export GOFLAGS="${GOFLAGS:-"-mod=vendor"}"

export PATH=$PATH:$GOPATH/bin
DONT_REBUILD_TEST_BINS="${DONT_REBUILD_TEST_BINS:-false}"

if [ "$DONT_REBUILD_TEST_BINS" == "false" ] || [ -f ./cnf-tests/bin/mirror ]; then
  echo "Building mirror ... "
  go build -o ./bin/mirror -tags strictfipsruntime mirror/mirror.go
fi

pushd submodules/cluster-node-tuning-operator
echo "Building latency tools ... "
make dist-latency-tests
popd
