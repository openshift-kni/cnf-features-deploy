Table of Contents
=================

* [The Conformance Tests ](#the-conformance-tests)
* [Running the tests](#running-the-tests)
* [Test run configuration environment variables](#test-run-configuration-environment-variables)
* [List of environment variables used in the tests](#list-of-environment-variables-used-in-the-tests)
* [Test Reports](#test-reports)
  * [JUnit test output](#junit-test-output)
  * [Test Failure Report](#test-failure-report)
* [Gatekeeper](#gatekeeper)

    
# The Conformance Tests 

The conformance test suites verify that the operators we maintain are working properly on CNF-enabled clusters. The suites are "validationsuite", "configsuite", and "cnftests". The underlying Ginkgo test suites run by each suite can be found in the `TESTS_PATHS` variable in `hack/common.sh` in the root directory.  

To prevent dependency conflicts, we use git submodules to store external repositories: `cluster-node-tuning-operator`, `metallb-operator`, and `sriov-network-operator`, in the path `cnf-tests/submodules`.


To specify the desired commit of the external repositories, we expose the following environment variables:

```
export METALLB_OPERATOR_TARGET_COMMIT?=main
export SRIOV_NETWORK_OPERATOR_TARGET_COMMIT?=main
export CLUSTER_NODE_TUNING_OPERATOR_TARGET_COMMIT?=main
```

And to set all three of them to a specfic release, use `TARGET_RELEASE`.

## Running the Tests

We invoke the tests using different Makefile targets, for example: `functests-on-ci`.

We invoke the tests using the Ginkgo CLI tool.

## Test run configuration environment variables

**KUBECONFIG**: The path to the kubeconfig for the cluster.  
**SKIP_TESTS**: Translates to the ginkgo skip flag.  
**FEATURES** and **FOCUS_TESTS**: Translates to the ginkgo focus flag.  
**GINKGO_PARAMS**: Provide any additional valid ginkgo parameters.  
**EXTERNAL_SUITES**: Specifies which external test suites to run; options include: (integration/metallb/nto/sriov/nto-performance)  
**TESTS_REPORTS_PATH**: Path where the test reports will be stored.  
**FAIL_FAST**: Set to "true" to enable the ginkgo `--fail-fast` flag.  
**TEST_SUITES**: : Set this if you want to focus on a specific suite; options are: (validationsuite/configsuite/cnftests)  

## List of environment variables used in the tests

- DPDK_TEST_NAMESPACE - dpdk tests namespace override
- SCTP_TEST_NAMESPACE - sctp tests namespace override
- SRIOV_OPERATOR_NAMESPACE - sriov operator namespace override
- PTP_OPERATOR_NAMESPACE - ptp operator namespace override
- OVS_QOS_TEST_NAMESPACE - ovs_qos tests namespace override
- NODES_SELECTOR - selector for limiting the nodes on which tests are executed
- SRIOV_WAITING_TIME - timout in minutes for sriov configuration to become stable before each sriov tests
- PERF_TEST_PROFILE - performance profile name override
- ROLE_WORKER_CNF - cnf tests worker pool override
- SCTPTEST_HAS_NON_CNF_WORKERS - no cnf worker in sctp tests, some sctp tests will be skipped if enabled
- CLEAN_PERFORMANCE_PROFILE - disable performance profile cleanup for faster tests
- PERFORMANCE_PROFILE_MANIFEST_OVERRIDE - performance profile manifest override
- IPERF3_BITRATE_OVERRIDE - set a maximum bitrate for iperf3 to use in ovs_qos tests
- SKIP_LOCAL_RESOURCES - use default test resource of dependant test suites, using hardcoded defaults instead, needed to successfuly run the metallb e2e tests

## Test Reports

The tests produce two kinds of outputs, which are stored in the directory specified by `TESTS_REPORTS_PATH`:

```
.
├── cnftests-junit.xml
├── junit_cnftests.xml
├── junit_setup.xml
├── junit_validation.xml
├── metallb_failure_report.log
│   └── metallb_MetalLB_deploy_should_have_frr-k8s_pods_in_running_state
│       ├── crs.log
│       ├── nodes.log
│       ├── openshift-metallb-system-pods_logs.log
│       └── openshift-metallb-system-pods_specs.log
├── setup_junit.xml
└── validation_junit.xml
```

### JUnit test output

A JUnit-compliant XML file is produced for each test suite: `junit_cnftests.xml`, `junit_setup.xml`, `junit_validation.xml`.

### Test Failure Report

A report containing information about the cluster state (and resources) for troubleshooting is produced for each failed test. For example, if a test fails in the metallb suite, the logs are dumped into the following file: `metallb_failure_report.log`.

## Gatekeeper

Refer [here](GATEKEEPER.md) for instructions on further gatekeeper testing.
