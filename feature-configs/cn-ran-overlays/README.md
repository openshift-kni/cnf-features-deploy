# cn-ran-overlays

This folder contains configurations for 5G DU / CU
CNF integration

## Overview

The [`all-in-one`](all-in-one) directory contains the Kustomize profile for deployment of DU integration features, namely:
- SCTP MachineConfig patch
- Performance addon operator and DU performance profile
- PTP operator and slave profile
- SR-IOV operator and associated profiles

## Deployment

1. Make sure your nodes are labeled as required and MCP is created. If your cluster is brand new, this can be done by cnf-features-deploy/hack/setup-test-cluster.sh
2. Deploy using the usual project tools:
  
  `FEATURES_ENVIRONMENT=cn-ran-overlays FEATURES=all-in-one make feature-deploy`

3. Wait for the iterations to complete
