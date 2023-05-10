#!/bin/bash
set -e

. $(dirname "$0")/common.sh

export PATH=$PATH:$GOPATH/bin
export failed=false
export failures=()
export GINKGO_PARAMS=${GINKGO_PARAMS:-'-vv --show-node-events'}

#env variables needed for the containerized version
export TEST_POD_IMAGES_REGISTRY="${TEST_POD_IMAGES_REGISTRY:-quay.io/openshift-kni/}"
export TEST_POD_CNF_TEST_IMAGE="${TEST_POD_CNF_TEST_IMAGE:-cnf-tests:4.12}"
export TEST_POD_DPDK_TEST_IMAGE="${TEST_POD_DPDK_TEST_IMAGE:-dpdk:4.12}"

export TEST_EXECUTION_IMAGE=$TEST_POD_IMAGES_REGISTRY$TEST_POD_CNF_TEST_IMAGE
export SCTPTEST_HAS_NON_CNF_WORKERS="${SCTPTEST_HAS_NON_CNF_WORKERS:-true}"
# In CI we don't care about cleaning the profile, because we may either throw the cluster away
# or need to run the tests again. In both cases the execution will be faster without deleting the profile.
export CLEAN_PERFORMANCE_PROFILE="false"

# Latency tests env variables
export LATENCY_TEST_RUN=${LATENCY_TEST_RUN:-false}

export IS_OPENSHIFT="${IS_OPENSHIFT:-true}"

# Map for the suites' junit report names
declare -A JUNIT_REPORT_NAME=( ["configsuite"]="junit_setup.xml" ["cnftests"]="junit_cnftests.xml"  ["validationsuite"]="junit_validation.xml")

echo "Running local tests"


if [ "$DONT_FOCUS" == true ]; then
	echo "per-feature tests disabled, all tests but the one skipped will be executed"
elif [ "$FEATURES" == "" ]; then
	echo "No FEATURES provided"
  exit 1
else
  FOCUS="--focus="$(echo "$FEATURES" | tr ' ' '|')
  if [ "$FOCUS_TESTS" != "" ]; then
    FOCUS="--focus="$(echo "$FOCUS_TESTS" | tr ' ' '|')
  fi
  echo "Focusing on $FOCUS"
fi

if [ "$SKIP_TESTS" != "" ]; then
	SKIP="--skip="$(echo "$SKIP_TESTS" | tr ' ' '|')
	echo "Skip set, skipping $SKIP"
fi

export SUITES_PATH=cnf-tests/bin

go build -o cnf-tests/bin/junit-merger cnf-tests/testsuites/pkg/junit-merger/junit-merger.go

TEST_SUITES=${TEST_SUITES:-"validationsuite configsuite cnftests"}
suites=( $TEST_SUITES )

for suite in "${suites[@]}"; do
  if [ "$DISCOVERY_MODE" == "true" ] &&  [ "$suite" == "configsuite" ]; then
      echo "Discovery mode enabled, skipping setup"
      continue
  fi
# If the EXTERNAL_SUITES variable is empty, run all the external suites for suite
  if [[ -z "$EXTERNAL_SUITES" ]]; then
    case $suite in
      "configsuite")
        external_suites=("nto")
        ;;
      "validationsuite")
        external_suites=("integration" "metallb")
        ;;
      "cnftests")
        external_suites=("integration" "metallb" "nto-performance" "nto-latency" "ptp" "sriov")
        ;;
      *)
        echo "Invalid suite name: $suite"
        exit 1
        ;;
    esac
  else
    external_suites=($EXTERNAL_SUITES)
  fi

  for external_suite in "${external_suites[@]}"; do
    TEST_PATH="${TESTS_PATHS[$suite $external_suite]}"
    if [[ -n "$TEST_PATH" ]]; then
      EXEC_TESTS="ginkgo -tags=validationtests,e2etests $FAIL_FAST $SKIP $FOCUS $GINKGO_PARAMS --junit-report="$suite-$external_suite-junit.xml" --output-dir="${TESTS_REPORTS_PATH}" $TEST_PATH"
      if ! $EXEC_TESTS; then
        failed=true
        failures+=( "Tier 2 tests for $external_suite" )
      fi
    else
      echo "Invalid external suite name for suite $suite: $external_suite"
      exit 1
    fi
  done

  if [[ -n "$TESTS_REPORTS_PATH" ]]; then
   cnf-tests/bin/junit-merger -o "${TESTS_REPORTS_PATH}"/"${JUNIT_REPORT_NAME[$suite]}" "${TESTS_REPORTS_PATH}"/"$suite-"*"-junit.xml"
   rm "${TESTS_REPORTS_PATH}"/"$suite"-*-junit.xml
  fi
done

set +e
# JUnit reports are written in `junit_cnftests.xml`, `junit_validation.xml` and `junit_setup.xml` but some CI systems
# still relies on old used paths. Symlinking them for backward compatibility.
# Note that Prow CI searches for `junit*.xml` files while rendering tests in job page.
# Allow it to fail in case a report is missing because the script was invoked for a specific test suite.
ln -sf "$TESTS_REPORTS_PATH/junit_setup.xml" "$TESTS_REPORTS_PATH/setup_junit.xml"
ln -sf "$TESTS_REPORTS_PATH/junit_cnftests.xml" "$TESTS_REPORTS_PATH/cnftests-junit.xml"
ln -sf "$TESTS_REPORTS_PATH/junit_validation.xml" "$TESTS_REPORTS_PATH/validation_junit.xml"

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

if $failed; then
  echo "[WARN] Tests failed:"
  for failure in "${failures[@]}"; do
    echo "$failure"
  done;
  exit 1
fi
