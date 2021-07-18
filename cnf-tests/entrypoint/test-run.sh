#!/bin/bash
set -e
# Setting -e is fine as we want both config and validiation to succeed
# before running the "real" tests.

TEST_SUITES=${TEST_SUITES:-"flakesuite"}
SUITES_PATH="${SUITES_PATH:-~/usr/bin}"

suites=( $TEST_SUITES )
if [ "$IMAGE_REGISTRY" != "" ] && [[ "$IMAGE_REGISTRY" != */ ]]; then
    export IMAGE_REGISTRY="$IMAGE_REGISTRY/"
fi

for suite in "${suites[@]}"; do
    if [ "$DISCOVERY_MODE" == "true" ] &&  [ "$suite" == "configsuite" ]; then
        echo "Discovery mode enabled, skipping setup"
        continue
    fi
    echo running "$SUITES_PATH/$suite" "$@"
    "$SUITES_PATH/$suite" "$@"
done
