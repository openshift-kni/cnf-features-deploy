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

if ! which ginkgo; then
	echo "Installing ginkgo tool from vendor"
	go install -mod=vendor github.com/onsi/ginkgo/v2/ginkgo
fi

mkdir -p bin/cnftests
mkdir -p bin/configsuite
mkdir -p bin/validationsuite

export current_path=$PWD

function build_and_move_suite {
  suite=$1
  feature=$2
  source=$3
  tags=$4
  target_dir=${current_path}/bin/${suite}/
  if [ "$DONT_REBUILD_TEST_BINS" == "false" ] || [ ! -f "$target" ]; then
    cd $source
    ginkgo build -mod=mod --tags="$tags" 
    output=$(ls "./"*".test")
    mv $output $target_dir/${feature}.test
    cd $current_path
  fi
}

build_and_move_suite "cnftests" "integration" "testsuites/e2esuite"
build_and_move_suite "cnftests" "metallb" "submodules/metallb-operator/test/e2e/functional" "e2etests"
build_and_move_suite "cnftests" "sriov" "submodules/sriov-network-operator/test/conformance"
build_and_move_suite "cnftests" "nto-performance" "submodules/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/1_performance"
build_and_move_suite "cnftests" "nto-latency" "submodules/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/4_latency"
build_and_move_suite "cnftests" "ptp" "submodules/ptp-operator/test/conformance/serial"

build_and_move_suite "configsuite" "nto" "submodules/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/0_config"

build_and_move_suite "validationsuite" "cluster" "testsuites/validationsuite"
build_and_move_suite "validationsuite" "metallb" "submodules/metallb-operator/test/e2e/validation" "validationtests"

if [ "$DONT_REBUILD_TEST_BINS" == "false" ] || [ -f ./cnf-tests/bin/mirror ]; then
  go build -o ./bin/mirror mirror/mirror.go
fi
