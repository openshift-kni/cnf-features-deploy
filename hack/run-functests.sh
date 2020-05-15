#!/bin/bash

pushd .
cd "$(dirname "$0")/.."

which go
if [ $? -ne 0 ]; then
  echo "No go command available"
  exit 1
fi

GOPATH="${GOPATH:-~/go}"

export CONTAINER_MGMT_CLI="${CONTAINER_MGMT_CLI:-docker}"
export PATH=$PATH:$GOPATH/bin
export TESTS_REPORTS_PATH="${TESTS_REPORTS_PATH:-/tmp/artifacts/}"
export failed=false
export failures=()

export TEST_POD_IMAGES_REGISTRY="${TEST_POD_IMAGES_REGISTRY:-quay.io/openshift-kni/}"
export TEST_POD_CNF_TEST_IMAGE="${TEST_POD_CNF_TEST_IMAGE:-cnf-tests:4.5}"
export TEST_POD_DPDK_TEST_IMAGE="${TEST_POD_DPDK_TEST_IMAGE:-dpdk:4.5}"

export CNF_TESTS_IMAGE=$TEST_POD_IMAGES_REGISTRY$TEST_POD_CNF_TEST_IMAGE

if [ "$FEATURES" == "" ]; then
	echo "[ERROR]: No FEATURES provided"
	exit 1
fi

echo "Running local tests"
FOCUS=$(echo "$FEATURES" | tr ' ' '|')
echo "Focusing on $FOCUS"
export SUITES_PATH=cnf-tests/bin

mkdir -p "$TESTS_REPORTS_PATH"

if [ "$TESTS_IN_CONTAINER" == "true" ]; then
  cp -f "$KUBECONFIG" _cache/kubeconfig
  echo "Running dockerized version via $CNF_TESTS_IMAGE"
  EXEC_TESTS="$CONTAINER_MGMT_CLI run -v $(pwd)/_cache/:/kubeconfig:Z -v $TESTS_REPORTS_PATH:/reports:Z -e CNF_TESTS_IMAGE=$TEST_POD_CNF_TEST_IMAGE -e DPDK_TESTS_IMAGE=$TEST_POD_DPDK_TEST_IMAGE -e IMAGE_REGISTRY=$TEST_POD_IMAGES_REGISTRY -e KUBECONFIG=/kubeconfig/kubeconfig $CNF_TESTS_IMAGE /usr/bin/test-run.sh \
       -ginkgo.focus $FOCUS -junit /reports/ -report /reports/"
else
  hack/build-test-bin.sh
  EXEC_TESTS="cnf-tests/test-run.sh -ginkgo.focus=$FOCUS -junit $TESTS_REPORTS_PATH -report $TESTS_REPORTS_PATH"
fi

if ! $EXEC_TESTS; then
  failed=true
  failures+=( "Tier 2 tests for $FEATURES" )
fi

if [[ ! $RUN_ORIGIN_TESTS ]]; then
  EXTERNALS="$FEATURES"
else
  echo "[INFO] Adding origin tests to be run"
  EXTERNALS="$FEATURES origintests"
fi

echo "Running external tests"
for feature in $EXTERNALS; do
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

function finish {
    popd
}
trap finish EXIT
