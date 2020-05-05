# The CNF Tests image

The [CNF tests image](https://quay.io/openshift-kni/cnf-tests) is a containerized version of the CNF conformance test suite.
It's intended to be run against a cluster where all the components required for running CNF workloads are installed.

This include:

- Targeting a machine config pool to which the machines to be tested belong to
- Enabling sctp via machine config
- Having the Performance Addon Operator installed
- Having the SR-IOV operator installed
- Having the PTP operator installed

## Running the tests

The test entrypoint is `/usr/bin/test-run.sh`. It runs both a "setup" test set and the real conformance test suite.
The bare minimum requirement is to provide it a kubeconfig file and it's related $KUBECONFIG environment variable, mounted through a volume.

Assuming the file is in the current folder, the command for running the test suite is:

```bash
    docker run -v $(pwd)/:/kubeconfig -e KUBECONFIG=/kubeconfig/kubeconfig quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh
```

This allows your kubeconfig file to be consumed from inside the running container.

## Pre-requisites

Some tests require a pre-existing machine config pool to append their changes to. This needs to be created on the cluster before running the tests.

The default worker pool is `worker-cnf` and can be created with the following manifest:

```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfigPool
metadata:
  name: worker-cnf
  labels:
    machineconfiguration.openshift.io/role: worker-cnf
spec:
  machineConfigSelector:
    matchExpressions:
      - {
          key: machineconfiguration.openshift.io/role,
          operator: In,
          values: [worker-cnf, worker],
        }
  paused: false
  nodeSelector:
    matchLabels:
      node-role.kubernetes.io/worker-cnf: ""
```

The worker pool name can be overridden via the `ROLE_WORKER_CNF` variable.

```bash
    docker run -v $(pwd)/:/kubeconfig -e KUBECONFIG=/kubeconfig/kubeconfig -e ROLE_WORKER_CNF=custom-worker-pool quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh
```

Please note that currently not all tests run selectively on the nodes belonging to the pool.

## Gingko Parameters

The test suite is built upon the ginkgo bdd framework.
This means that it accept parameters for filtering or skipping tests.

To filter a set of tests, the -ginkgo.focus parameter can be added:

```bash
    docker run -v $(pwd)/:/kubeconfig -e KUBECONFIG=/kubeconfig/kubeconfig quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh -ginkgo.focus="performance|sctp"
```

### Available features

The set of available features to filter are:

- performance
- sriov
- ptp
- sctp
- dpdk

### Dry Run

To run in dry-run mode:

```bash
    docker run -v $(pwd)/:/kubeconfig -e KUBECONFIG=/kubeconfig/kubeconfig quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh -ginkgo.dryRun -ginkgo.v
```

## Disconnected Mode

The CNF tests image support running tests in a disconnected cluster, meaning a cluster that is not able to reach outer registries.
This is done in two steps:

### Mirroring the images to a custom registry accessible from the cluster

A `mirror` executable is shipped in the image to provide the input required by oc to mirror the images needed to run the tests to a local registry. **Please note that it's mandatory for the registry to end with a `/`**.

The following command

```bash
    docker run -v $(pwd)/:/kubeconfig -e KUBECONFIG=/kubeconfig/kubeconfig quay.io/openshift-kni/cnf-tests /usr/bin/mirror -registry my.local.registry:5000/ |  oc image mirror -f -
```

can be run from an intermediate machine that has access both to the cluster and to quay.io over the Internet.

### Instruct the tests to consume those images from a custom registry

This is done by setting the IMAGE_REGISTRY environment variable:

```bash
    docker run -v $(pwd)/:/kubeconfig -e KUBECONFIG=/kubeconfig/kubeconfig -e IMAGE_REGISTRY my.local.registry:5000/ quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh
```

## Test Reports

The tests have two kind of outputs

### JUnit test output

A junit compliant xml is produced by passing the `--junit` parameter together with the path where the report is dumped:

```bash
    docker run -v $(pwd)/:/kubeconfig -v $(pwd)/junitdest:/path/to/junit -e KUBECONFIG=/kubeconfig/kubeconfig quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh --junit /path/to/junit
```

### Test Failure Report

A report with informations about the cluster state (and resources) for troubleshooting can be produced by passing the `--report` parameter together with the path where the report is dumped:

```bash
    docker run -v $(pwd)/:/kubeconfig -v $(pwd)/reportdest:/path/to/report -e KUBECONFIG=/kubeconfig/kubeconfig quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh --report /path/to/report
```
