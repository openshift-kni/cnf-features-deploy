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
The deployment is done using the usual project tools:

`FEATURES_ENVIRONMENT=cn-ran-overlays FEATURES=all-in-one make feature-deploy`