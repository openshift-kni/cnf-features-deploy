#!/bin/bash
. $(dirname "$0")/common.sh
set +e

if [ "$FEATURES_ENVIRONMENT" == "" ]; then
	echo "[ERROR]: No FEATURES_ENVIRONMENT provided"
	exit 1
fi

if [ "$FEATURES" == "" ]; then
	echo "[ERROR]: No FEATURES provided"
	exit 1
fi

ATTEMPTS=0
MAX_ATTEMPTS=15
all_ready=false
export TEST_SUITES="validationsuite"
export FAIL_FAST="-ginkgo.failFast"
export DONT_REBUILD_TEST_BINS=true

until $all_ready || [ $ATTEMPTS -eq $MAX_ATTEMPTS ]
do
    # we only care about the latest run failures, removing the logs from the previous
    # run
    rm -rf "$TESTS_REPORTS_PATH"
    echo "running tests"
    if hack/run-functests.sh; then
        echo "succeeded"
        all_ready=true
    else    
        echo "failed, retrying"
    fi
    (( ATTEMPTS++ ))
done

if ! $all_ready; then 
    echo "Timed out waiting for features to be ready"
    oc get nodes
    exit 1
fi
