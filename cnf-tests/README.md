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

## Image Parameters

The tests can use a different image in the test.
There are two images used by the tests that can be changed using the following environment variables.

```bash
# CNF_TESTS_IMAGE
# DPDK_TESTS_IMAGE
```

For example, to change the `CNF_TESTS_IMAGE` with a custom registry run the following command

```bash
docker run -v $(pwd)/:/kubeconfig -e KUBECONFIG=/kubeconfig/kubeconfig -e CNF_TESTS_IMAGE="custom-cnf-tests-image:latests" quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh
```

## Instruct the tests to consume those images from a custom registry

This is done by setting the IMAGE_REGISTRY environment variable:

```bash
    docker run -v $(pwd)/:/kubeconfig -e KUBECONFIG=/kubeconfig/kubeconfig -e IMAGE_REGISTRY="my.local.registry:5000/" -e CNF_TESTS_IMAGE="custom-cnf-tests-image:latests" quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh
```

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

Then follow the instructions above about overriding the registry used to fetch the images.

### Mirroring a different set of images

The mirror command tries to mirror the u/s images by default.
This can be overridden by passing a file with the following format to the image:

```json
[
    {
        "registry": "public.registry.io:5000",
        "image": "imageforcnftests:4.5"
    },
    {
        "registry": "public.registry.io:5000",
        "image": "imagefordpdk:4.5"
    }
]
```

And by passing it to the mirror command, for example saving it locally as `images.json`.
With the following command, the local path is mounted in `/kubeconfig` inside the container and that can be passed to the mirror command.

```bash
    docker run -v $(pwd)/:/kubeconfig -e KUBECONFIG=/kubeconfig/kubeconfig quay.io/openshift-kni/cnf-tests /usr/bin/mirror --registry "my.local.registry:5000/" --images "/kubeconfig/images.json" |  oc image mirror -f -
```

### Mirroring to the internal registry

The instructions are based on the official OpenShift documentation about [exposing the registry](https://docs.openshift.com/container-platform/4.4/registry/securing-exposing-registry.html).

```bash
oc patch configs.imageregistry.operator.openshift.io/cluster --patch '{"spec":{"defaultRoute":true}}' --type=merge
```

Fetch the registry endpoint:

```bash
REGISTRY=$(oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}')
```

Create a namespace for exposing the images:

```bash
oc create ns cnftests
```

Make that imagestream available to all the namespaces used for tests. This is required to allow the tests namespaces to fetch the images from the cnftests imagestream.

```bash
oc policy add-role-to-user system:image-puller system:serviceaccount:sctptest:default --namespace=cnftests
oc policy add-role-to-user system:image-puller system:serviceaccount:cnf-features-testing:default --namespace=cnftests
oc policy add-role-to-user system:image-puller system:serviceaccount:performance-addon-operators-testing:default --namespace=cnftests
oc policy add-role-to-user system:image-puller system:serviceaccount:dpdk-testing:default --namespace=cnftests
oc policy add-role-to-user system:image-puller system:serviceaccount:sriov-conformance-testing:default --namespace=cnftests
```

Retrieve the docker secret name and auth token:

```bash
SECRET=$(oc -n cnftests get secret | grep builder-docker | awk {'print $1'}
TOKEN=$(oc -n cnftests get secret $SECRET -o jsonpath="{.data['\.dockercfg']}" | base64 -d | jq '.["image-registry.openshift-image-registry.svc:5000"].auth')
```

Write a `dockerauth.json` like:

```bash
echo "{\"auths\": { \"$REGISTRY\": { \"auth\": $TOKEN } }}" > dockerauth.json
```

Do the mirroring:

```bash
docker run -v $(pwd)/:/kubeconfig -e KUBECONFIG=/kubeconfig/kubeconfig quay.io/openshift-kni/cnf-tests /usr/bin/mirror -registry $REGISTRY/cnftests |  oc image mirror --insecure=true -a=$(pwd)/dockerauth.json -f -
```

Run the tests:

```bash
docker run -v $(pwd)/:/kubeconfig -e KUBECONFIG=/kubeconfig/kubeconfig -e IMAGE_REGISTRY=image-registry.openshift-image-registry.svc:5000/cnftests cnf-tests-local:latest /usr/bin/test-run.sh
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

### A note on podman

When executing podman as non root (and non privileged) mounting paths may fail with "permission denied" errors. In order to make it work, `:Z` needs to be appended to the volumes creation (like `-v $(pwd)/:/kubeconfig:Z`) in order to allow podman to do the proper selinux relabelling (more details [here](https://github.com/containers/libpod/issues/3683#issuecomment-517239831)).

### Running on 4.4

The tests in the suite are compatible with OpenShift 4.4, except the following ones:

```bash
[test_id:28466][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Should contain configuration injected through openshift-node-performance profile 
[test_id:28467][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Should contain configuration injected through the openshift-node-performance profile
```

Skipping them can be done by adding the `-ginkgo.skip "28466|28467" parameter`

### Using a single performance profile

The resource needed by the dpdk tests are higher than those required by the performance test suite. To make the execution quicker, the performance profile used by tests can be overridden using one that serves also the dpdk test suite.

To do that, a profile like the following one can be mounted inside the container, and the performance tests can be instructed to deploy it.

```yaml
apiVersion: performance.openshift.io/v1alpha1
kind: PerformanceProfile
metadata:
  name: performance
spec:
  cpu:
    isolated: "0-15"
    reserved: "0-7"
  hugepages:
    defaultHugepagesSize: "1G"
    pages:
    - size: "1G"
      count: 16
      node: 0
  realTimeKernel:
    enabled: true
  nodeSelector:
    node-role.kubernetes.io/worker-cnf: ""
```

To override the performance profile used, the manifest must be mounted inside the container and the tests must be instructed by setting the `PERFORMANCE_PROFILE_MANIFEST_OVERRIDE` as follows:

```bash
docker run -v $(pwd)/:/kubeconfig:Z -e KUBECONFIG=/kubeconfig/kubeconfig -e PERFORMANCE_PROFILE_MANIFEST_OVERRIDE=/kubeconfig/manifest.yaml quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh
```
