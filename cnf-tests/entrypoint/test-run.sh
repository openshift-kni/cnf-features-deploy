#!/bin/bash

LATENCY_TEST_RUN="${LATENCY_TEST_RUN:-true}"

if [ "$IMAGE_REGISTRY" != "" ] && [[ "$IMAGE_REGISTRY" != */ ]]; then
    export IMAGE_REGISTRY="$IMAGE_REGISTRY/"
fi

echo running "/usr/bin/latency-e2e.test"
LATENCY_TEST_RUN="$LATENCY_TEST_RUN" "/usr/bin/latency-e2e.test"
