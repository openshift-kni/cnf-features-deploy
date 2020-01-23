#!/bin/bash

export TESTS_REPO=https://github.com/openshift-kni/performance-addon-operators
export TESTS_LOCATION=/tmp/performance-operators

external-tests/clone_repo.sh
cd $TESTS_LOCATION
ROLE_WORKER_RT=worker-cnf PERF_TEST_PROFILE=performance make functests-only
