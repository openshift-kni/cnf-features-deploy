# Overview

This repo contains example kustomize configs used to installed openshift features required for CNF workloads and a e2e functional test suite used to verify CNF related features.

## Contributing kustomize configs

All kustomize configs should be entirely declarative in nature. This means no bash plugin modules performing imperative tasks. Features should be installed simply by posting manifests to the cluster. After posting manifests, determining when the cluster has converged on those manifests successully should be observable.

## Usage

### Prerequisites

- You need a running OCP 4.4 (or later) cluster and a valid KUBECONFIG.
- You need at least one node with the `node-role.kubernetes.io/worker-cnf=""` label and a `MachineConfigPool` matching `worker-cnf` machine configurations
- You need to install `jq` (a command line tool for parsing JSON) on the local machine.

You can run `make setup-test-cluster` to have the first two (or the first in case of only one) workers labeled as `worker-cnf` and to have the `MachineConfigPool` created.

### Configuring

All the Makefile rules depend on two environment variable, either for deploying, waiting and choosing what tests to run.

##### FEATURES

e.g. `FEATURES="sctp ptp sriov"`, drives what features are going to be deployed using kustomize, and what tests are going to be run.

The current default values is `"sctp performance"`

##### FEATURES_ENVIRONMENT

i.e. `FEATURES_ENVIRONMENT=demo` determines the kustomization layer that will be used to deploy the choosen features.

The current default value is `e2e-gcp`

### Deployment

For each feature choosen via `FEATURES` we expect to have a layer either in [feature-configs/deploy](feature-configs/deploy) or in [feature-configs/$FEATURES_ENVIRONMENT](feature-configs/demo).

- run `FEATURES_ENVIRONMENT=demo make feature-deploy`.  
  This will try to apply all manifests in a loop until all deployments succeeded, or until it runs into a timeout.
- optionally run `FEATURES_ENVIRONMENT=demo make feature-wait` to be notified of when the features are deployed.

### Testing

We expect to have a section of [the test suite](functests/test_suite_test.go) named after each feature we want to test (for example [sctp](functests/sctp/sctp.go) named after the sctp feature).

External tests are consumed as dependencies and ran as part of this same suite.

### origin-tests

Verifies behavior of an OCP cluster by running remote tests against the cluster API that exercise functionality.
These tests may be disruptive.

Running a dockerized version of origin-tests from [quay.io/openshift/origin-tests](quay.io/openshift/origin-tests).
The full test suite can be found at [https://github.com/openshift/openshift-tests](https://github.com/openshift/openshift-tests).

- run `ORIGIN_TESTS_FILTER=openshift/conformance/serial make origin-tests`.
  The current default values is `openshift/conformance/parallel`

### Dockerized version

A dockerized version of CNF tests is available at [quay.io/openshift-kni/cnf-tests](quay.io/openshift-kni/cnf-tests).
For more details on how to use it, please check the [corresponding docs](cnf-tests/README.md).

### Example RAN profile

An example RAN profile is provided in [feature-configs/cn-ran-overlays](feature-configs/cn-ran-overlays)
