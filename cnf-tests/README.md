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

Assuming the kubeconfig file is in the current folder, the command for running the test suite is:

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

## Gingko Parameters

The test suite is built upon the ginkgo bdd framework.
This means that it accept parameters for filtering or skipping tests.

To filter a set of tests, the -ginkgo.focus parameter can be added:

```bash
    docker run -v $(pwd)/:/kubeconfig -e KUBECONFIG=/kubeconfig/kubeconfig quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh -ginkgo.focus="performance|sctp"
```

### Integration tests note

There is a particular test (`[sriov] SCTP integration`) that requires both SR-IOV and SCTP. Given the selective nature of the focus parameter, that test is triggered by only placing the `sriov` matcher. If the tests are executed against a cluster where SR-IOV is installed but SCTP is not, adding the `-ginkgo.skip=SCTP` parameter will make the tests to skip it.

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

This is done in two steps: performing the mirroring, and instructing the tests to consume the images from a custom registry.

### Mirroring the images to a custom registry accessible from the cluster

A `mirror` executable is shipped in the image to provide the input required by oc to mirror the images needed to run the tests to a local registry.

The following command

```bash
    docker run -v $(pwd)/:/kubeconfig -e KUBECONFIG=/kubeconfig/kubeconfig quay.io/openshift-kni/cnf-tests /usr/bin/mirror -registry my.local.registry:5000/ |  oc image mirror -f -
```

can be run from an intermediate machine that has access both to the cluster and to quay.io over the Internet.

Then follow the instructions above about overriding the registry used to fetch the images.

### Instruct the tests to consume those images from a custom registry

This is done by setting the IMAGE_REGISTRY environment variable:

```bash
    docker run -v $(pwd)/:/kubeconfig -e KUBECONFIG=/kubeconfig/kubeconfig -e IMAGE_REGISTRY="my.local.registry:5000/" -e CNF_TESTS_IMAGE="custom-cnf-tests-image:latests" quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh
```

### Mirroring to the cluster internal registry

OpenShift Container Platform provides a built in container image registry which runs as a standard workload on the cluster.

The instructions are based on the official OpenShift documentation about [exposing the registry](https://docs.openshift.com/container-platform/4.4/registry/securing-exposing-registry.html).

Gain external access to the registry by exposing it with a route:

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
docker run -v $(pwd)/:/kubeconfig -e KUBECONFIG=/kubeconfig/kubeconfig -e IMAGE_REGISTRY=image-registry.openshift-image-registry.svc:5000/cnftests quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh
```

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

## Running tests in discovery mode

The tests need to perform an environment configuration every time they are executed. This involves items such as creating Sriov Node Policies, Performance Profiles or PtpProfiles. Allowing the tests to configure an already configured cluster may affect the functionality of the cluster. Also changes to configuration items such as Sriov Node Policy might result in the environment being temporarily unavailable until the configuration change is processed.

Discovery mode allows to validate the functionality of a cluster without altering its configuration. Existing environment configuration will be used for the tests. The tests will attempt to find the configuration items needed, and use those items to execute the tests. If resources needed to run a specific test are not found, the test will be skipped (providing an appropriate message to the user). After the tests are finished, no cleanup of the preconfigured configuration items is done, and the test environment can immediately be used for another test run.

Some configuration items are still created by the tests. These are specific items needed for a test to run, such as for example a Sriov Network. These configuration items are created in custom namespaces and are cleaned up after the tests are executed.

An additional bonus is a reduction in test run times. As the configuration items are already there, no time is needed for environment configuration and stabilization.

### Enabling discovery mode

To enable discovery mode the tests must be instructed by setting the `DISCOVERY_MODE` environemnt variable as follows:

```bash
  docker run -v $(pwd)/:/kubeconfig:Z -e KUBECONFIG=/kubeconfig/kubeconfig -e DISCOVERY_MODE=true quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh
```

### Environement configuration prerequisites required for discovery mode

#### SRIOV tests

Most SRIOV test require the following resources:

- SriovNetworkNodePolicy
- at least one one with the resource specified by SriovNetworkNodePolicy being allocatable (a resource count of at least 5 is considered sufficient)

Some tests have additional requirements:

- an unused device on the node with available policy resource (with link state DOWN and not a bridge slave)
- a SriovNetworkNodePolicy with a MTU value of 9000

#### DPDK tests

The DPDK related tests require:

- a PerformanceProfile
- a SRIOV policy
- a node with resources available for the SRIOV policy and available with the PerformanceProfile node selector

#### PTP tests

- a slave PtpConfig (ptp4lOpts="-s" ,phc2sysOpts="-a -r")
- a node with a label matching the slave PtpConfig

#### SCTP tests

- SriovNetworkNodePolicy
- a node matchin both the SriovNetworkNodePolicy and a MachineConfig which enables SCTP

#### Performance operator tests

Various tests have different requirements. Some of them:
- a PerformanceProfile
- a PerformanceProfile having profile.Spec.CPU.Isolated = 1
- a PerformanceProfile having profile.Spec.RealTimeKernel.Enabled == true
- a node with no hugepages usage

### Limiting the nodes used during tests.

The nodes on which the tests will be executed can limited by means of specifying a `NODES_SELECTOR` environment variable. Any resources created by the test will then be limited to the specified nodes.

```bash
  docker run -v $(pwd)/:/kubeconfig:Z -e KUBECONFIG=/kubeconfig/kubeconfig -e NODES_SELECTOR=node-role.kubernetes.io/worker-cnf quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh
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

## Reducing test running time

### Using a single performance profile

The resource needed by the dpdk tests are higher than those required by the performance test suite. To make the execution quicker, the performance profile used by tests can be overridden using one that serves also the dpdk test suite.

To do that, a profile like the following one can be mounted inside the container, and the performance tests can be instructed to deploy it.

```yaml
apiVersion: performance.openshift.io/v1
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

### Disabling the performance profile cleanup

When not running in discovery mode, the suite cleans up all the artifacts / configurations created. This includes the performance profile.

When deleting the performance profile, the machine config pool is modified, and this includes nodes rebooting. After a new iteration, a new profile will be created. This causes long test cycles between runs.

To make it quicker, a `CLEAN_PERFORMANCE_PROFILE="false"` can be set to instruct the tests not to clean the performance profile. In this way the next iteration won't need to create it and wait for it to be applied.

```bash
docker run -v $(pwd)/:/kubeconfig:Z -e KUBECONFIG=/kubeconfig/kubeconfig -e CLEAN_PERFORMANCE_PROFILE="false" quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh
```

## Troubleshooting

The cluster must be reached from within the container. One thing to verify that is by running:

```bash
    docker run -v $(pwd)/:/kubeconfig -e KUBECONFIG=/kubeconfig/kubeconfig quay.io/openshift-kni/cnf-tests oc get nodes
```

If this does not work, it may be for several reason, spanning across dns, mtu size, firewall to mention some.

## Impacts on the Cluster

Depending on the feature, running the test suite might have different impact on the cluster.
In general, only the `sctp` tests do not change the cluster configuration. All the other features have impacts on the configuration in a way or another.

### SCTP

SCTP tests just run different pods on different nodes to check connectivity. The impacts on the cluster are related to running simple pods on two nodes.

### SR-IOV

SR-IOV tests require changes in the SR-IOV network configuration, and SR-IOV tests create and destroy different types of configuration.

This may have an impact if existing SR-IOV network configurations are already installed on the cluster, because there may be conflicts depending on the priority of such configurations.

At the same time, the result of the tests may be affected by already existing configurations.

### PTP

PTP tests apply a ptp configuration to a set of nodes of the cluster. As per SR-IOV, this may conflict with any existing PTP configuration already in place, with unpredictable results.

### Performance

Performance tests apply a performance profile to the cluster. The effect of this is changes in the node configuration, reserving CPUs, allocating memory hugepages, setting the kernel packages to be realtime.

If an existing profile named `performance` is already available on the cluster, the tests do not deploy it.

### DPDK

DPDK relies on both `performance` and `SR-IOV` features, so the test suite both configure a `performance profile` and `SR-IOV` networks, so the impacts are the same described above.

### Cleaning Up

After running the test suite, all the dangling resources are cleaned up.
