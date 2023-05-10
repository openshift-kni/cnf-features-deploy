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

all_ready=false
export TEST_SUITES="validationsuite"
export FAIL_FAST="--fail-fast"
export DONT_REBUILD_TEST_BINS=true
export TIMEOUT="${TIMEOUT:-5400}" # results in 90 minutes timeout

echo "[INFO]: Wait $TIMEOUT seconds for features to be ready"

start_time=$(date +%s)
until $all_ready
do
    # we only care about the latest run failures, removing the logs from the previous
    # run
    rm -rf "$TESTS_REPORTS_PATH"
    echo "running tests"
    if hack/run-functests.sh; then
        echo "succeeded"
        all_ready=true
    else    
        time_now=$(date +%s)
        elapsed=$(( time_now - start_time ))
        time_left=$(( TIMEOUT - elapsed ))
        if [ $time_left -le 0 ]; then
            break
        fi
        echo "failed, retrying. $time_left seconds left till timeout"
    fi
done

if ! $all_ready; then 
    echo "Timed out waiting for features to be ready"
    oc get nodes
    exit 1
fi
