#!/bin/bash

if [ "$IMAGE_REGISTRY" != "" ] && [[ "$IMAGE_REGISTRY" != */ ]]; then
    export IMAGE_REGISTRY="$IMAGE_REGISTRY/"
fi

echo running "/usr/bin/latency-e2e.test"
"/usr/bin/latency-e2e.test"
