#!/bin/bash

. $(dirname "$0")/common.sh

export PATH=$PATH:$GOPATH/bin
export failed=false
export failures=()
export GINKGO_PARAMS=${GINKGO_PARAMS:-'-ginkgo.vv -ginkgo.show-node-events'}

#env variables needed for the containerized version
export TEST_POD_IMAGES_REGISTRY="${TEST_POD_IMAGES_REGISTRY:-quay.io/openshift-kni/}"
export TEST_POD_CNF_TEST_IMAGE="${TEST_POD_CNF_TEST_IMAGE:-cnf-tests:4.14}"
export TEST_POD_DPDK_TEST_IMAGE="${TEST_POD_DPDK_TEST_IMAGE:-dpdk:4.14}"

export TEST_EXECUTION_IMAGE=$TEST_POD_IMAGES_REGISTRY$TEST_POD_CNF_TEST_IMAGE
export SCTPTEST_HAS_NON_CNF_WORKERS="${SCTPTEST_HAS_NON_CNF_WORKERS:-true}"
# In CI we don't care about cleaning the profile, because we may either throw the cluster away
# or need to run the tests again. In both cases the execution will be faster without deleting the profile.
export CLEAN_PERFORMANCE_PROFILE="false"

# Latency tests env variables
export LATENCY_TEST_RUN=${LATENCY_TEST_RUN:-false}

export IS_OPENSHIFT="${IS_OPENSHIFT:-true}"

echo "Running local tests"


if [ "$DONT_FOCUS" == true ]; then
	echo "per-feature tests disabled, all tests but the one skipped will be executed"
elif [ "$FEATURES" == "" ]; then
	echo "No FEATURES provided"
  exit 1
else
  FOCUS="-ginkgo.focus="$(echo "$FEATURES" | tr ' ' '|')
  if [ "$FOCUS_TESTS" != "" ]; then
    FOCUS="-ginkgo.focus="$(echo "$FOCUS_TESTS" | tr ' ' '|')
  fi
  echo "Focusing on $FOCUS"
fi

if [ "$SKIP_TESTS" != "" ]; then
	SKIP="-ginkgo.skip="$(echo "$SKIP_TESTS" | tr ' ' '|')
	echo "Skip set, skipping $SKIP"
fi

export SUITES_PATH=cnf-tests/bin

mkdir -p "$TESTS_REPORTS_PATH"
if [ ! -f "cnf-tests/bin/j2html" ] && [ "$JUNIT_TO_HTML" == "true" ]; then
    go build -o cnf-tests/bin/j2html cnf-tests/testsuites/pkg/j2html/j2html.go
fi


if [ "$TESTS_IN_CONTAINER" == "true" ]; then
  cp -f "$KUBECONFIG" _cache/kubeconfig
  echo "Running dockerized version via $TEST_EXECUTION_IMAGE"

  features="$(echo "$FEATURES" | tr ' ' '|')"

  env_vars="-e CLEAN_PERFORMANCE_PROFILE=false \
  -e CNF_TESTS_IMAGE=$TEST_POD_CNF_TEST_IMAGE \
  -e DPDK_TESTS_IMAGE=$TEST_POD_DPDK_TEST_IMAGE \
  -e IMAGE_REGISTRY=$TEST_POD_IMAGES_REGISTRY \
  -e KUBECONFIG=/kubeconfig/kubeconfig \
  -e SCTPTEST_HAS_NON_CNF_WORKERS=$SCTPTEST_HAS_NON_CNF_WORKERS \
  -e TEST_SUITES=$TEST_SUITES \
  -e IS_OPENSHIFT=$IS_OPENSHIFT \
  -e FEATURES=$features \
  -e OO_INSTALL_NAMESPACE=$OO_INSTALL_NAMESPACE"

  # add latency tests env variable to the cnf-tests container
  if [ "$LATENCY_TEST_RUN" == "true" ];then
    env_vars="$env_vars \
    -e LATENCY_TEST_RUN=$LATENCY_TEST_RUN \
    -e LATENCY_TEST_RUNTIME=$LATENCY_TEST_RUNTIME \
    -e LATENCY_TEST_DELAY=$LATENCY_TEST_DELAY \
    -e OSLAT_MAXIMUM_LATENCY=$OSLAT_MAXIMUM_LATENCY \
    -e CYCLICTEST_MAXIMUM_LATENCY=$CYCLICTEST_MAXIMUM_LATENCY \
    -e HWLATDETECT_MAXIMUM_LATENCY=$HWLATDETECT_MAXIMUM_LATENCY \
    -e MAXIMUM_LATENCY=$MAXIMUM_LATENCY"
  fi

  EXEC_TESTS="$CONTAINER_MGMT_CLI run \
  -v $(pwd)/_cache/:/kubeconfig:Z \
  -v $TESTS_REPORTS_PATH:/reports:Z \
  --network host \
  ${env_vars} \
  $TEST_EXECUTION_IMAGE /usr/bin/test-run.sh $FAIL_FAST $SKIP $FOCUS $GINKGO_PARAMS -junit /reports/ -report /reports/"
else
  cnf-tests/hack/build-test-bin.sh
  EXEC_TESTS="cnf-tests/entrypoint/test-run.sh $FAIL_FAST $SKIP $FOCUS $GINKGO_PARAMS -junit $TESTS_REPORTS_PATH -report $TESTS_REPORTS_PATH"
fi

reports="cnftests_failure_report.log setup_failure_report.log validation_failure_report.log"
for report in $reports; do 
  if [[ -f "$TESTS_REPORTS_PATH/$report" || -d "$TESTS_REPORTS_PATH/$report" ]]; then  
    tar -czf "$TESTS_REPORTS_PATH/$report.""$(date +"%Y-%m-%d_%T")".gz -C "$TESTS_REPORTS_PATH" "$report" --remove-files --force-local
  fi
done

# JUnit reports are written in `junit_cnftests.xml`, `junit_validation.xml` and `junit_setup.xml` but some CI systems
# still relies on old used paths. Symlinking them for backward compatibility.
# Note that Prow CI searches for `junit*.xml` files while rendering tests in job page.
ln -sf "$TESTS_REPORTS_PATH/junit_setup.xml" "$TESTS_REPORTS_PATH/setup_junit.xml"
ln -sf "$TESTS_REPORTS_PATH/junit_cnftests.xml" "$TESTS_REPORTS_PATH/cnftests-junit.xml"
ln -sf "$TESTS_REPORTS_PATH/junit_validation.xml" "$TESTS_REPORTS_PATH/validation_junit.xml"


if ! $EXEC_TESTS; then
  failed=true
  failures+=( "Tier 2 tests for $FEATURES" )
fi

echo "Running external tests"
for feature in $FEATURES; do
  test_entry_point=external-tests/${feature}/test.sh
  if [[ ! -f $test_entry_point ]]; then
    echo "[INFO] Feature '$feature' does not have external tests entry point"
    continue
  fi
  echo "[INFO] Running external tests for $feature"
  set +e
  if ! $test_entry_point; then
    failures+=( "Tier 1 tests for $feature" )
    failed=true
  fi
  set -e
  if [[ -f /tmp/artifacts/unit_report.xml ]]; then
    mv /tmp/artifacts/unit_report.xml "/tmp/artifacts/unit_report_external_${feature}.xml"
  fi
done

# if env var is set, convert junit report to html
if [ "$JUNIT_TO_HTML" == "true" ]; then
  if [ ! -f "$TESTS_REPORTS_PATH/cnftests-junit.xml" ]; then
    echo "No cnftests junit report found, skipping conversion to html"
  else
    cnf-tests/bin/j2html < "$TESTS_REPORTS_PATH/cnftests-junit.xml" > "$TESTS_REPORTS_PATH/cnftests.html"
  fi
  if [ ! -f "$TESTS_REPORTS_PATH/validation_junit.xml" ]; then
    echo "No validationsuite junit report found, skipping conversion to html"
  else
    cnf-tests/bin/j2html < "$TESTS_REPORTS_PATH/validation_junit.xml" > "$TESTS_REPORTS_PATH/validation.html"
  fi
  if [ ! -f "$TESTS_REPORTS_PATH/setup_junit.xml" ]; then
    echo "No configsuite junit report found, skipping conversion to html"
  else
    cnf-tests/bin/j2html < "$TESTS_REPORTS_PATH/setup_junit.xml" > "$TESTS_REPORTS_PATH/setup.html"
  fi
fi

if $failed; then
  echo "[WARN] Tests failed:"
  for failure in "${failures[@]}"; do
    echo "$failure"
  done;
  exit 1
fi
