#!/bin/bash

set -e

if ! which go; then
  echo "No go command available"
  exit 1
fi

GOPATH="${GOPATH:-~/go}"
GOFLAGS="${GOFLAGS:-"-mod=vendor"}"

export PATH=$PATH:$GOPATH/bin

if ! which gingko; then
	echo "Downloading ginkgo tool"
	go install github.com/onsi/ginkgo/ginkgo
fi

ginkgo build ./functests
mkdir -p cnf-tests/bin
mv ./functests/functests.test ./cnf-tests/bin/cnftests
