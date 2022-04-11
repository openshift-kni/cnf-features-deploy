Table of Contents
=================

* [The CNF Tests image](#the-cnf-tests-image)
  * [Running the tests](#running-the-tests)
    * [Running latency test](#running-latency-test)
  * [Running only latency test](#running-only-latency-test)
  * [Pre\-requisites](#pre-requisites)
  * [Image Parameters](#image-parameters)
  * [Gingko Parameters](#gingko-parameters)
    * [Integration tests note](#integration-tests-note)
    * [Available features](#available-features)
    * [Dry Run](#dry-run)
  * [Disconnected Mode](#disconnected-mode)
    * [Mirroring the images to a custom registry accessible from the cluster](#mirroring-the-images-to-a-custom-registry-accessible-from-the-cluster)
    * [Instruct the tests to consume those images from a custom registry](#instruct-the-tests-to-consume-those-images-from-a-custom-registry)
    * [Mirroring to the cluster internal registry](#mirroring-to-the-cluster-internal-registry)
    * [Mirroring a different set of images](#mirroring-a-different-set-of-images)
  * [Running tests in discovery mode](#running-tests-in-discovery-mode)
    * [Enabling discovery mode](#enabling-discovery-mode)
    * [Environement configuration prerequisites required for discovery mode](#environement-configuration-prerequisites-required-for-discovery-mode)
      * [SRIOV tests](#sriov-tests)
      * [DPDK tests](#dpdk-tests)
      * [PTP tests](#ptp-tests)
      * [SCTP tests](#sctp-tests)
      * [XT_U32 tests](#xt_u32-tests)
      * [Performance operator tests](#performance-operator-tests)
      * [Container-mount-namespace tests](#container-mount-namespace-tests)
    * [Limiting the nodes used during tests\.](#limiting-the-nodes-used-during-tests)
  * [Test Reports](#test-reports)
    * [JUnit test output](#junit-test-output)
    * [Test Failure Report](#test-failure-report)
    * [A note on podman](#a-note-on-podman)
    * [Running on 4\.4](#running-on-44)
  * [Reducing test running time](#reducing-test-running-time)
    * [Using a single performance profile](#using-a-single-performance-profile)
    * [Disabling the performance profile cleanup](#disabling-the-performance-profile-cleanup)
  * [Running in single node cluster](#running-in-single-node-cluster)
  * [Troubleshooting](#troubleshooting)
  * [Impacts on the Cluster](#impacts-on-the-cluster)
    * [SCTP](#sctp)
    * [XT_U32](#xt_u32)
    * [SR\-IOV](#sr-iov)
    * [PTP](#ptp)
    * [Performance](#performance)
    * [DPDK](#dpdk)
    * [Container-mount-namespace](#container-mount-namespace)
    * [Cleaning Up](#cleaning-up)

# The CNF Tests image

The [CNF tests image](https://quay.io/openshift-kni/cnf-tests) is a containerized version of the CNF conformance test suite.
It's intended to be run against a cluster where all the components required for running CNF workloads are installed.

This include:

- Targeting a machine config pool to which the machines to be tested belong to
- Enabling sctp via machine config
- Enabling xt_u32 via machine config
- Having the SR-IOV operator installed
- Having the PTP operator installed
- Enabling the contain-mount-namespace mode via machine config

## Running the tests

The test entrypoint is `/usr/bin/test-run.sh`. It runs both a "setup" test set and the real conformance test suite.
The bare minimum requirement is to provide it with a kubeconfig file and its related $KUBECONFIG environment variable, mounted through a volume.

Assuming the kubeconfig file is in the current folder, the command for running the test suite is:

```bash
docker run -v $(pwd)/:/kubeconfig -e KUBECONFIG=/kubeconfig/kubeconfig quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh
```

This allows your kubeconfig file to be consumed from inside the running container.

### Running latency test

As part of the performance test suite, you have the latency test available that by default disabled.
To enable the latency test, you should provide a set of additional environment variables.

- LATENCY_TEST_RUN - should be true, if you want to run the latency test(default false).
- LATENCY_TEST_RUNTIME - how long do you want to run the latency test binary(default 5m).
- LATENCY_TEST_CPUS - the amount of CPUs the pod which run the latency test should request(default all isolated CPUs - 1).
- OSLAT_MAXIMUM_LATENCY - what the maximum latency do you expect to have during the `oslat` run, the value should be greater than 0(default -1, that means the latency check will not run).
- CYCLICTEST_MAXIMUM_LATENCY - what the maximum latency do you expect to have during the `cyclictest` run, the value should be greater than 0(default -1, that means the latency check will not run).
- HWLATDETECT_MAXIMUM_LATENCY - what the maximum latency do you expect to have during the `hwlatdetect` run, the value should be greater than 0(default -1, that means the latency check will not run).
- MAXIMUM_LATENCY - what the maximum latency do you expect to have during all the latency tests run, the value should be greater than 0(default -1, that means the latency check will not run).

The command to running the test suite with the latency test:

```bash
docker run -v $(pwd)/:/kubeconfig \ 
-e KUBECONFIG=/kubeconfig/kubeconfig \
-e LATENCY_TEST_RUN=true \
-e LATENCY_TEST_RUNTIME=600 \
-e OSLAT_MAXIMUM_LATENCY=20 \
-e CYCLICTEST_MAXIMUM_LATENCY=20 \
-e HWLATDETECT_MAXIMUM_LATENCY=20 \
quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh
```
Or use the unified ```MAXIMUM_LATENCY``` for all the tests
(In case both provided, the specific variables will have precedence over the unified one).

## Running only latency test

To run only the configuration, and the latency test, you should provide the `ginkgo.focus` parameter, and
the environment variable that contains the name of the performance profile that should be tested:

```bash
docker run --rm -v $KUBECONFIG:/kubeconfig \
-e KUBECONFIG=/kubeconfig \
-e LATENCY_TEST_RUN=true \
-e LATENCY_TEST_RUNTIME=600 \
-e OSLAT_MAXIMUM_LATENCY=20 \
-e CYCLICTEST_MAXIMUM_LATENCY=20 \
-e HWLATDETECT_MAXIMUM_LATENCY=20 \
-e PERF_TEST_PROFILE=<performance_profile_name> \
quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh -ginkgo.focus="\[performance\]\[config\]|\[performance\]\ Latency\ Test"
```
Or use the unified ```MAXIMUM_LATENCY``` for all the tests
(In case both provided, the specific variables will have precedence over the unified one).

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
This means that it accepts parameters for filtering or skipping tests.

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
- xt_u32
- dpdk
- container-mount-namespace

A detailed list of the tests can be found [here](./TESTLIST.md).

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

Some configuration items are still created by the tests. These are specific items needed for a test to run, such as Sriov Networks. These configuration items are created in custom namespaces and are cleaned up after the tests are executed.

An additional bonus is a reduction in test run times. As the configuration items are already there, no time is needed for environment configuration and stabilization.

**Note:** The validation step is performed even when running in discovery mode. This means that if a test run is meant to validate a given feature but the feature is not installed, the whole suite will fail, on par with regular mode. As an example, if the SR-IOV operator is not deployed on the cluster, but discovery mode is ran to validate SR-IOV (or all the features, including SR-IOV), the whole suite will fail. To overcome this, the test must filter only the features available on the cluster.

### Enabling discovery mode

To enable discovery mode the tests must be instructed by setting the `DISCOVERY_MODE` environemnt variable as follows:

```bash
  docker run -v $(pwd)/:/kubeconfig:Z -e KUBECONFIG=/kubeconfig/kubeconfig -e DISCOVERY_MODE=true quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh
```

### Environement configuration prerequisites required for discovery mode

#### SRIOV tests

Most SRIOV test require the following resources:

- SriovNetworkNodePolicy
- at least one with the resource specified by SriovNetworkNodePolicy being allocatable (a resource count of at least 5 is considered sufficient)

Some tests have additional requirements:

- an unused device on the node with available policy resource (with link state DOWN and not a bridge slave)
- an SriovNetworkNodePolicy with an MTU value of 9000

#### DPDK tests

The DPDK related tests require:

- a PerformanceProfile
- an SRIOV policy
- a node with resources available for the SRIOV policy and available with the PerformanceProfile node selector

#### PTP tests

- a slave PtpConfig (ptp4lOpts="-s" ,phc2sysOpts="-a -r")
- a node with a label matching the slave PtpConfig

#### SCTP tests

- SriovNetworkNodePolicy
- a node matching both the SriovNetworkNodePolicy and a MachineConfig which enables SCTP

#### XT_U32 tests

- a node with a MachineConfig which enables XT_U32

#### Performance operator tests

Various tests have different requirements. Some of them:
- a PerformanceProfile
- a PerformanceProfile that has more than one CPU (profile.Spec.CPU.Isolated) allocated
- a PerformanceProfile that has profile.Spec.RealTimeKernel.Enabled == true
- a node with no hugepages usage

#### Container-mount-namespace tests

- a node with a MachineConfig which enables container-mount-namespace mode

### Limiting the nodes used during tests.

The nodes on which the tests will be executed can be limited by means of specifying a `NODES_SELECTOR` environment variable. Any resources created by the test will then be limited to the specified nodes.

```bash
  docker run -v $(pwd)/:/kubeconfig:Z -e KUBECONFIG=/kubeconfig/kubeconfig -e NODES_SELECTOR=node-role.kubernetes.io/worker-cnf quay.io/openshift-kni/cnf-tests /usr/bin/test-run.sh
```

### Clean Up - Known Limitations

Temporary objects (such as namespaces, pods, ..) are deleted when the suite finishes.

As a side effect, all the network policies / ptp configurations starting with "test-" are deleted.

## Test Reports

The tests have two kinds of outputs

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

Skipping them can be done by adding the `-ginkgo.skip "28466|28467"` parameter

## Reducing test running time

### Using a single performance profile

The resources needed by the dpdk tests are higher than those required by the performance test suite. To make the execution quicker, the performance profile used by tests can be overridden using one that serves also the dpdk test suite.

To do that, a profile like the following one can be mounted inside the container, and the performance tests can be instructed to deploy it.

```yaml
apiVersion: performance.openshift.io/v1
kind: PerformanceProfile
metadata:
  name: performance
spec:
  cpu:
    isolated: "4-15"
    reserved: "0-3"
  hugepages:
    defaultHugepagesSize: "1G"
    pages:
    - size: "1G"
      count: 4
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

## Running in a single node cluster

Running tests on a single node cluster causes some limitations to be imposed:
- longer timeouts for certain tests
- some tests requiring multiple nodes are skipped

The longer timeouts concern mostly SRIOV and SCTP tests. Reconfiguration requiring node reboots cause a reboot of the whole environment including the openshift control plane, and hence takes longer to complete.
Some PTP tests requiring a master and a slave node are skipped.
No additional configuration is needed for this, the tests will check for the number of nodes at startup and adjust the tests behavior accordingly.

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

### XT_U32

XT_U32 tests just run pods on different nodes to check iptables rule that utilize xt_u32. The impacts on the cluster are related to running simple pods on two nodes.

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

### Container-mount-namespace

The validation test for container-mount-namespace mode only checks that the appropriate MachineConfig objects are present and active, and has no additional impact on the node.

### Cleaning Up

After running the test suite, all the dangling resources are cleaned up.

## List of environment variables used in the tests

- DPDK_TEST_NAMESPACE - dpdk tests namespace override
- SCTP_TEST_NAMESPACE - sctp tests namespace override
- XT_U32_TEST_NAMESPACE - xt-u32 tests namespace override
- SRIOV_OPERATOR_NAMESPACE - sriov operator namespace override
- PTP_OPERATOR_NAMESPACE - ptp operator namespace override
- OVS_QOS_TEST_NAMESPACE - ovs_qos tests namespace override
- DISCOVERY_MODE - discover mode switch
- NODES_SELECTOR - selector for limiting the nodes on which tests are executed
- SRIOV_WAITING_TIME - timout in minutes for sriov configuration to become stable before each sriov tests
- PERF_TEST_PROFILE - performance profile name override
- ROLE_WORKER_CNF - cnf tests worker pool override
- SCTPTEST_HAS_NON_CNF_WORKERS - no cnf worker in sctp tests, some sctp tests will be skipped if enabled
- XT_U32TEST_HAS_NON_CNF_WORKERS - no cnf worker in xt_u32 tests, some xt_u32 tests will be skipped if enabled
- CLEAN_PERFORMANCE_PROFILE - disable performance profile cleanup for faster tests
- PERFORMANCE_PROFILE_MANIFEST_OVERRIDE - performance profile manifest override
- IPERF3_BITRATE_OVERRIDE - set a maximum bitrate for iperf3 to use in ovs_qos tests
- SKIP_LOCAL_RESOURCES - use default test resource of dependant test suites, using hardcoded defaults instead, needed to successfuly run the metallb e2e tests

## Additional testing

### Gatekeeper

Refer [here](GATEKEEPER.md) for instructions on further gatekeeper testing.
