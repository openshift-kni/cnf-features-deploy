#!/bin/bash

# Check for updated worker-cnf node

oc wait mcp/worker-cnf --for condition=updated --timeout 1s


# Check for real-time kernel.

WORKER_NODE=$(oc get nodes --selector="node-role.kubernetes.io/worker-cnf" --output="name")

echo "worker-cnf node is: ${WORKER_NODE}"

oc describe "${WORKER_NODE}" | grep "Kernel Version:.*\.rt[0-9]*\..*\.x86_64"
