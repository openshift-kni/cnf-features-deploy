#!/bin/sh

# Wait for performance-addon-operator deployment to be ready
until ${OC_TOOL} -n openshift-performance-addon get deploy/performance-operator; do
    echo "[INFO]: get performance-operator deployment"
    sleep 10
done
${OC_TOOL} -n openshift-performance-addon wait deploy/performance-operator --for condition=Available --timeout 5m
