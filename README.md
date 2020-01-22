# Overview

This repo contains example kustomize configs used to installed openshift features required for CNF workloads and a e2e functional test suite used to verify cnf related features.

# Contributing kustomize configs

All kustomize configs should be entirely declarative in nature. This means no bash plugin modules performing imparative tasks. Features should be installed simply by posting manifests to the cluster. After posting manifests, determining when the cluster has converged on those manifests successully should be observable.

# Usage

## Prerequisites

- You need a running OCP 4.4 cluster and a valid KUBECONFIG.
- You need at least one node with the `node-role.kubernetes.io/worker-cnf=""` label and a `MachineConfigPool` matching `worker-cnf` machine configurations  
  Run `make setup-test-cluster` for adding it on the first two `worker` nodes of the cluster.

## Deployment

- run `FEATURES_ENVIRONMENT=demo make feature-deploy`.  
  This will try to apply all manifests in a loop until all deployments succeeded, or until it runs into a timeout.
