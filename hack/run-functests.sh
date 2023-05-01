#!/bin/bash

. $(dirname "$0")/common.sh

export PATH=$PATH:$GOPATH/bin
export failed=false
export failures=()
export GINKGO_PARAMS=${GINKGO_PARAMS:-'-ginkgo.vv -ginkgo.show-node-events'}

#env variables needed for the containerized version
export TEST_POD_IMAGES_REGISTRY="${TEST_POD_IMAGES_REGISTRY:-quay.io/openshift-kni/}"
export TEST_POD_CNF_TEST_IMAGE="${TEST_POD_CNF_TEST_IMAGE:-cnf-tests:4.12}"
export TEST_POD_DPDK_TEST_IMAGE="${TEST_POD_DPDK_TEST_IMAGE:-dpdk:4.12}"

export TEST_EXECUTION_IMAGE=$TEST_POD_IMAGES_REGISTRY$TEST_POD_CNF_TEST_IMAGE
export SCTPTEST_HAS_NON_CNF_WORKERS="${SCTPTEST_HAS_NON_CNF_WORKERS:-true}"
# In CI we don't care about cleaning the profile, because we may either throw the cluster away
# or need to run the tests again. In both cases the execution will be faster without deleting the profile.
export CLEAN_PERFORMANCE_PROFILE="false"

export IS_OPENSHIFT="${IS_OPENSHIFT:-true}"

export SUITES_PATH=cnf-tests/bin

# Map for the suites' junit report names
declare -A JUNIT_REPORT_NAME=( ["configsuite"]="junit_setup.xml" ["cnftests"]="junit_cnftests.xml"  ["validationsuite"]="junit_validation.xml")


if [[ -n "$TESTS_REPORTS_PATH" ]]; then
  mkdir -p "$TESTS_REPORTS_PATH"
  junit="-junit $TESTS_REPORTS_PATH"
  report="-report $TESTS_REPORTS_PATH"
fi

if [[ -z "$TEST_SUITES" ]]; then
  TEST_SUITES=("configsuite" "validationsuite" "cnftests")
else
  TEST_SUITES=($TEST_SUITES)
fi

if [[ -n "$FOCUS_TESTS" ]]; then
  FOCUS="-ginkgo.focus="$(echo "$FOCUS_TESTS" | tr ' ' '|')
fi

if [[ -n "$SKIP_TESTS" ]]; then
	SKIP="-ginkgo.skip="$(echo "$SKIP_TESTS" | tr ' ' '|')
fi

if [[ -n "$FAIL_FAST" ]]; then
  GINKGO_PARAMS="${GINKGO_PARAMS} --ginkgo.fail-fast"
fi

# Validate TEST_SUITES variable is valid
for SUITE in "${TEST_SUITES[@]}"; do
  case $SUITE in
    "configsuite")
      ;;
    "validationsuite")
      ;;
    "cnftests")
      ;;
    *)
      echo "Invalid suite name: $SUITE"
      exit 1
      ;;
  esac
done

mkdir -p "$TESTS_REPORTS_PATH"

# Pring the test run Configurations
echo "--------------Test Run Configurations--------------"
echo "Skip the following: $SKIP"
echo "Focus the following: $FOCUS"
echo "Run the following ginkgo params: $GINKGO_PARAMS"
echo "Reports path: $TESTS_REPORTS_PATH"
echo "---------------------------------------------------"

for SUITE in "${TEST_SUITES[@]}"; do
# If the FEATURES variable is empty, run all the features for suite
  if [[ -z "$FEATURES" ]]; then
    case $SUITE in
      "configsuite")
        SUITE_FEATURES=("nto")
        ;;
      "validationsuite")
        SUITE_FEATURES=("cluster" "metallb")
        ;;
      "cnftests")
        SUITE_FEATURES=("integration" "metallb" "nto" "ptp" "sriov")
        ;;
      *)
        echo "Invalid suite name: $SUITE"
        exit 1
        ;;
    esac
  else
    SUITE_FEATURES=($FEATURES)
  fi

  echo "Now running suite ($SUITE) with the feature(s) ${SUITE_FEATURES[*]}"
  for FEATURE in "${SUITE_FEATURES[@]}"; do
    for TEST_FILE in "$SUITES_PATH/$SUITE/$FEATURE"*".test"; do
      if [[ -f "$TEST_FILE" ]]; then
        EXEC_TESTS="$TEST_FILE $junit $report $FOCUS $SKIP $GINKGO_PARAMS"
        if ! $EXEC_TESTS; then
          failed=true
          failures+=( "Tier 2 tests for $FEATURE" )
        fi
      else
        echo "Invalid feature name for suite $SUITE: $FEATURE"
        exit 1
      fi
    done
  done

  if [[ -n "$TESTS_REPORTS_PATH" ]]; then
    junit-report-merger "${TESTS_REPORTS_PATH}"/"${JUNIT_REPORT_NAME[$SUITE]}" "${TESTS_REPORTS_PATH}"/*junit.xml
    rm "${TESTS_REPORTS_PATH}"/*junit.xml
  fi

done

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

if $failed; then
  echo "[WARN] Tests failed:"
  for failure in "${failures[@]}"; do
    echo "$failure"
  done;
  exit 1
fi
