#!/bin/bash

source "$(dirname "$0")/../setup.sh"

export TESTS_REPO=https://github.com/openshift/sriov-network-operator
export TESTS_LOCATION=/tmp/sriov-tests
export REMOTE_BRANCH=$SRIOV_TESTS_BRANCH

external-tests/clone_repo.sh
cd $TESTS_LOCATION
make test-e2e-conformance
