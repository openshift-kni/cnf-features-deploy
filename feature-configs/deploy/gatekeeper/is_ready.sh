#!/bin/sh

oc -n gatekeeper-system wait deployment/gatekeeper-controller-manager --for condition=available --timeout 1s
oc -n gatekeeper-system wait deployment/gatekeeper-audit --for condition=available --timeout 1s
