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

if ! which gingko; then
	echo "Downloading ginkgo tool"
	go install github.com/onsi/ginkgo/ginkgo
fi

mkdir -p cnf-tests/bin

function build_and_move_suite {
  suite=$1
  target=$2

  if [ "$DONT_REBUILD_TEST_BINS" == "false" ] || [ ! -f "$target" ]; then
    ginkgo build ./"$suite"
    mv ./"$suite"/"$suite".test "$target"
  fi
}

build_and_move_suite "functests" "./bin/cnftests"
build_and_move_suite "configsuite" "./bin/configsuite"
build_and_move_suite "validationsuite" "./bin/validationsuite"

if [ "$DONT_REBUILD_TEST_BINS" == "false" ] || [ -f ./cnf-tests/bin/mirror ]; then
  go build -o ./bin/mirror mirror/mirror.go
fi



