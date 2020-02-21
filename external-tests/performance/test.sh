#!/bin/bash

source "$(dirname "$0")/../setup.sh"

export TESTS_REPO=https://github.com/openshift-kni/performance-addon-operators
export TESTS_LOCATION=/tmp/performance-operators
export REMOTE_BRANCH=$PERF_OPERATOR_BRANCH

external-tests/clone_repo.sh
cd $TESTS_LOCATION
ROLE_WORKER_RT=worker-cnf PERF_TEST_PROFILE=performance make functests-only
