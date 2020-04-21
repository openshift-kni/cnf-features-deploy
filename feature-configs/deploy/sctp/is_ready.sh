#!/bin/sh

# no output means that the new machine config wasn't picked by MCO yet
if [ -z "$(oc get mcp worker-cnf -o jsonpath='{.spec.configuration.source[?(@.name=="load-sctp-module")].name}')" ]; then
    exit 1
fi

oc wait mcp/worker-cnf --for condition=updated --timeout 1s
