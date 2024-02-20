#!/bin/bash
set -e

. $(dirname "$0")/common.sh

export PATH=$PATH:$GOPATH/bin
export failed=false
export failures=()
export GINKGO_PARAMS=${GINKGO_PARAMS:-'-vv --show-node-events -timeout 6h'}

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

extract_test_suites() {
  local step_name="$1"
  local suite_names=()

  for key in "${!TESTS_PATHS[@]}"; do
    if [[ $key == "${step_name} "* ]]; then
      suite_names+=("${key#* }")
    fi
  done

  echo "${suite_names[@]}"
}

if ! which ginkgo; then
	echo "Installing ginkgo tool from vendor"
	go install -mod=vendor github.com/onsi/ginkgo/v2/ginkgo
fi

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

if [ ! -f "cnf-tests/bin/junit-merger" ]; then
  go build -o cnf-tests/bin/junit-merger cnf-tests/testsuites/pkg/junit-merger/junit-merger.go
fi
if [ ! -f "cnf-tests/bin/j2html" ] && [ "$JUNIT_TO_HTML" == "true" ]; then
    go build -o cnf-tests/bin/j2html cnf-tests/testsuites/pkg/j2html/j2html.go
fi
mkdir -p "$TESTS_REPORTS_PATH"

TEST_SUITES=${TEST_SUITES:-"validationsuite configsuite cnftests"}
steps=( $TEST_SUITES )
REPORTERS=""

for step in "${steps[@]}"; do
  if [ "$DISCOVERY_MODE" == "true" ] &&  [ "$suite" == "configsuite" ]; then
      echo "Discovery mode enabled, skipping setup"
      continue
  fi
# If the EXTERNAL_SUITES variable is empty, run all the external suites for the given step
  if [[ -z "$EXTERNAL_SUITES" ]]; then
    case $step in
      "configsuite")
        external_suites=($(extract_test_suites "configsuite"))
        ;;
      "validationsuite")
        external_suites=($(extract_test_suites "validationsuite"))
        ;;
      "cnftests")
        external_suites=($(extract_test_suites "cnftests"))
        ;;
      *)
        echo "Invalid step name: $step"
        exit 1
        ;;
    esac
  else
    external_suites=($EXTERNAL_SUITES)
  fi

  for external_suite in "${external_suites[@]}"; do
    TEST_PATH="${TESTS_PATHS[$step $external_suite]}"
    if [[ -n "$TESTS_REPORTS_PATH" ]]; then
      mkdir -p "$TESTS_REPORTS_PATH"
      REPORTERS="--junit-report=${step}-${external_suite}-junit.xml --output-dir=${TESTS_REPORTS_PATH}"
    fi
    if [[ -n "$TEST_PATH" ]]; then

      get_current_commit "$step" "$external_suite"
      echo "now testing $CURRENT_TEST"
      EXEC_TESTS="ginkgo -tags=validationtests,e2etests $FAIL_FAST $SKIP $FOCUS $GINKGO_PARAMS $REPORTERS $TEST_PATH -- -report=${TESTS_REPORTS_PATH}"
      if ! $EXEC_TESTS; then
        failed=true
        failures+=( "Tier 2 tests for $external_suite" )
      fi
    else
      echo "Invalid external suite name for step $step: $external_suite"
      exit 1
    fi
  done

  if [[ -n "$TESTS_REPORTS_PATH" ]]; then
   cnf-tests/bin/junit-merger -output "${TESTS_REPORTS_PATH}"/"${JUNIT_REPORT_NAME[$step]}" "${TESTS_REPORTS_PATH}"/"$step-"*"-junit.xml"
   rm "${TESTS_REPORTS_PATH}"/"$step"-*-junit.xml
  fi
done

set +e
# JUnit reports are written in `junit_cnftests.xml`, `junit_validation.xml` and `junit_setup.xml` but some CI systems
# still relies on old used paths. Symlinking them for backward compatibility.
# Note that Prow CI searches for `junit*.xml` files while rendering tests in job page.
# Allow it to fail in case a report is missing because the script was invoked for a specific step.
ln -sf "$TESTS_REPORTS_PATH/junit_setup.xml" "$TESTS_REPORTS_PATH/setup_junit.xml"
ln -sf "$TESTS_REPORTS_PATH/junit_cnftests.xml" "$TESTS_REPORTS_PATH/cnftests-junit.xml"
ln -sf "$TESTS_REPORTS_PATH/junit_validation.xml" "$TESTS_REPORTS_PATH/validation_junit.xml"

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
