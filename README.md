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

- optionally set `FOCUS_TESTS` variable also to run specific tests.

e.g. `FEATURES="sriov" FOCUS_TESTS="operator.Generic.SriovNetworkNodePolicy.Resource.Injector operator.Generic.SriovNetworkNodePolicy.VF.flags"`, drives sriov deployment using kustomize, and two sriov tests are going to be run.

##### FEATURES_ENVIRONMENT

i.e. `FEATURES_ENVIRONMENT=demo` determines the kustomization layer that will be used to deploy the choosen features.

The current default value is `e2e-gcp`

### Deployment

For each feature choosen via `FEATURES` we expect to have a layer either in [feature-configs/deploy](feature-configs/deploy) or in [feature-configs/$FEATURES_ENVIRONMENT](feature-configs/demo).

- run `FEATURES_ENVIRONMENT=demo make feature-deploy`.  
  This will try to apply all manifests in a loop until all deployments succeeded, or until it runs into a timeout.
- optionally run `FEATURES_ENVIRONMENT=demo make feature-wait` to be notified of when the features are deployed.

### Testing

We expect to have a section of [the test suite](cnf-tests/testsuites/e2esuite/test_suite_test.go) named after each feature we want to test (for example [sctp](cnf-tests/testsuites/e2esuite/sctp.go) named after the sctp feature).

External tests are consumed as dependencies and ran as part of this same suite.

### origin-tests

Verifies behavior of an OCP cluster by running remote tests against the cluster API that exercise functionality.
These tests may be disruptive.

Running a dockerized version of origin-tests from [quay.io/openshift/origin-tests](https://quay.io/openshift/origin-tests).
The full test suite can be found at [https://github.com/openshift/openshift-tests](https://github.com/openshift/openshift-tests).

- run `ORIGIN_TESTS_FILTER=openshift/conformance/serial make origin-tests`.
  The current default values is `openshift/conformance/parallel`

- optionally set `ORIGIN_TESTS_IN_DISCONNECTED_ENVIRONMENT=true` and `ORIGIN_TESTS_REPOSITORY=test.repository.com:5000/origin-tests` to run origin-tests in a disconnected environment.

- optionally run `ORIGIN_TESTS_REPOSITORY=test.repository.com:5000/origin-tests make mirror-origin-tests` to mirror all the required test images to a container image repository.

- optionally run `ORIGIN_TESTS_REPOSITORY=test.repository.com:5000/origin-tests make origin-tests-disconnected-environment` to mirror all the required test images to a container image repository and source required test images from your repository when running origin-tests in a disconnected environment.

### Custom RPMs
The custom RPMs utility allows to install RPMs from an external source on an OCP node.
RPMS_SRC (RPMs download source URL) must be provided.
REMOVE_PACKAGES should be set when installing RT kernel RPMs to override the installed kernel packages (no need to set it when replacing a regular kernel with a different regular kernel or a RT kernel with a different RT kernel).
RPMS_NODE_ROLE is optional and defaults to `node-role.kubernetes.io/worker`.

- run `RPMS_SRC="http://test.download.com/example1.rpm http://test.download.com/example2.rpm" make custom-rpms`.  
  This will install all the RPMs listed in RPMS_SRC on the selected nodes.

- optionally run `RPMS_SRC="http://test.download.com/rt-kernel-package1.rpm http://test.download.com/rt-kernel-package1.rpm http://test.download.com/rt-kernel-package3.rpm" REMOVE_PACKAGES="kernel kernel-core kernel-modules kernel-modules-extra" make custom-rpms` to install RT kernel RPMs listed in RPMS_SRC on the selected nodes and override the installed kernel packages listed in REMOVE_PACKAGES.

### The CNF-Tests Container (Latency Tests)

A dockerized version of the Node Tuning Operator latency test suite available at [quay.io/openshift-kni/cnf-tests](https://quay.io/openshift-kni/cnf-tests).
For more details on how to use it, please check the [official docs](https://docs.openshift.com/container-platform/4.14/scalability_and_performance/cnf-performing-platform-verification-latency-tests.html).

### The Conformance tests

The conformance test suites verify that the operators we maintain are working properly on CNF enabled cluster. Nightly upstream ci jobs are using those tests, triggered from the openshift/release ci-operator [link](https://github.com/openshift/release/blob/master/ci-operator/step-registry/telco5g/cnf/tests/telco5g-cnf-tests-commands.sh). For more details, see [cnf-tests/README.md](cnf-tests/README.md)

### Zero-Touch Provisioning for RAN

Zero-touch provisioning enables a gitops-based flow for deploying and configuring OpenShift for RAN applications.

For an overview, see [ztp/gitops-subscriptions/argocd/README.md](ztp/gitops-subscriptions/argocd/README.md)

## Release Branching

This repository follows the same version numbering and release branching schedule as OpenShift: https://docs.ci.openshift.org/docs/architecture/branching/

### Branching

1. Before creating a new `release-x.y` branch ensure the new `x.y`version is reflected in the following sources:
   - `cnf-tests/Dockerfile.openshift`:
     1. Check image tags used.
     2. `OCP_VERSION`.

   - `cnf-tests/.konflux/Dockerfile`:
     1. Check image tags used.
     2. `OCP_VERSION`.
     3. Check the tags in the strings of `RUN sed` commands.

    - `cnf-tests/mirror/images.json`
    - `cnf-tests/testsuites/pkg/images/images.go`
    - `hack/common.sh`

2. If any sources require updates, create a pull request against the master branch with the necessary changes. Merge this PR before proceeding to the next steps.
2. Create a new `release-x.y` branch and push it to git
3. Advance the CI configuration in openshift/release to create new lanes for the release branch and move the master along to the next release version number
    - Example: https://github.com/openshift/release/pull/26172
4. Change the ZTP templates examples label `du-profile: latest` under [gitops-subscriptions/argocd/example](https://github.com/openshift-kni/cnf-features-deploy/tree/master/ztp/gitops-subscriptions/argocd/example) to match the branch release-x.y
    - PGT Example: https://github.com/openshift-kni/cnf-features-deploy/pull/1063
    - SiteConfig Example: https://github.com/openshift-kni/cnf-features-deploy/pull/1064
5. Pin all midstream branches to the new release branch, and create new midstream branches for the next release
   - [cnf-tests](https://code.engineering.redhat.com/gerrit/admin/repos/cnf-tests)
   - [ztp-site-generate](https://code.engineering.redhat.com/gerrit/admin/repos/ztp-site-generate)
6. Create a PR to the master branch updating the references from the first step to version `x.y+1`