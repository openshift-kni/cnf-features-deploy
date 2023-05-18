#!/bin/bash
set -e
# Setting -e is fine as we want both config and validiation to succeed
# before running the "real" tests.

LATENCY_TEST_RUN="${LATENCY_TEST_RUN:-true}"
DISCOVERY_MODE="${DISCOVERY_MODE:-true}"
FEATURES="${FEATURES:-performance}"

if [ "$IMAGE_REGISTRY" != "" ] && [[ "$IMAGE_REGISTRY" != */ ]]; then
    export IMAGE_REGISTRY="$IMAGE_REGISTRY/"
fi

echo running "/usr/bin/latency-e2e.test"
LATENCY_TEST_RUN="$LATENCY_TEST_RUN" DISCOVERY_MODE="$DISCOVERY_MODE" FEATURES="$FEATURES" "/usr/bin/latency-e2e.test"
