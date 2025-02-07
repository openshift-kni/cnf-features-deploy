#!/bin/bash

DISCOVERY_MODE="${DISCOVERY_MODE:-true}"

if [ "$IMAGE_REGISTRY" != "" ] && [[ "$IMAGE_REGISTRY" != */ ]]; then
    export IMAGE_REGISTRY="$IMAGE_REGISTRY/"
fi

echo running "/usr/bin/latency-e2e.test"
DISCOVERY_MODE="$DISCOVERY_MODE" "/usr/bin/latency-e2e.test" "$@"
