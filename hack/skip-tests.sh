#!/bin/bash

. $(dirname "$0")/common.sh

SKIP_DIR=test-overrides

add_skip_test() {
    local extra_skip_file=$SKIP_DIR/$1
    if [[ -e $extra_skip_file ]]; then
        local additional_skip_tests=$(<$extra_skip_file)
        if [[ -n $additional_skip_tests ]]; then
            debug "Found additional skip tests in $extra_skip_file: $additional_skip_tests"
            SKIP_TESTS="$SKIP_TESTS $additional_skip_tests"
        fi
    fi
}

if [[ -n $FEATURES && -n $FEATURES_ENVIRONMENT ]]; then
    debug "Checking for extra tests to skip based on $FEATURES_ENVIRONMENT/$FEATURES"
    debug "Incoming skip list: $SKIP_TESTS"

    # Check for a base skip override for the group of features
    add_skip_test $FEATURES_ENVIRONMENT/skip
    # And check for feature-specific skip overrides
    for feature in $FEATURES; do
        add_skip_test $FEATURES_ENVIRONMENT/$feature/skip
    done

    # Remove any duplicates
    SKIP_TESTS=$(echo $(echo -e "${SKIP_TESTS// /\\n}" | sort -u))
    debug "Computed skip list: $SKIP_TESTS"
fi

echo $SKIP_TESTS
