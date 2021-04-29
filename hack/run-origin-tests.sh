#!/bin/bash

. $(dirname "$0")/origin-tests-common.sh

skopeo_workdir=$(mktemp -d)
function finish_skopeo {
  rm -rf ${skopeo_workdir}
}
trap finish_skopeo EXIT

ORIGIN_TESTS_REPORTS_PATH="${ORIGIN_TESTS_REPORTS_PATH:-/tmp/artifacts/}"

ORIGIN_TESTS_IN_DISCONNECTED_ENVIRONMENT="${ORIGIN_TESTS_IN_DISCONNECTED_ENVIRONMENT:-false}"

ORIGIN_TESTS_FILTER="${ORIGIN_TESTS_FILTER:-openshift/conformance/parallel}"
CLUSTER_PROVIDER="${CLUSTER_PROVIDER:-}"

failed=false

if [ "$ORIGIN_TESTS_IN_DISCONNECTED_ENVIRONMENT" == "true" ]; then

  # A container image repository to retrieve test images from.
  # All the required test images should be mirrored to this repository.
  if [ "$ORIGIN_TESTS_REPOSITORY" == "" ]; then
    echo "[ERROR]: No ORIGIN_TESTS_REPOSITORY provided, can't run in disconnected mode"
    exit 1
  fi

  flags="--from-repository $ORIGIN_TESTS_REPOSITORY"
fi

echo "Running $ORIGIN_TESTS_FILTER tests"

# The cluster provider is the infrastructure provider (azure, aws, etc.).
# Can be null when running on baremetal
if [ -n "$CLUSTER_PROVIDER" ]; then
  echo "Provider: $CLUSTER_PROVIDER"
  printf -v test_prov "%q" $CLUSTER_PROVIDER
  flags="$flags --provider $test_prov"
fi

mkdir -p "$ORIGIN_TESTS_REPORTS_PATH"

mkdir -p _cache/
cp -f "$KUBECONFIG" _cache/kubeconfig

if [ "$TESTS_IN_CONTAINER" == "true" ]; then
  EXEC_TESTS="$CONTAINER_MGMT_CLI run -v $(pwd)/_cache/:/kubeconfig:Z -v $ORIGIN_TESTS_REPORTS_PATH:/reports:Z \
  -e KUBECONFIG=/kubeconfig/kubeconfig -it $ORIGIN_TESTS_IMAGE \
  /usr/bin/openshift-tests run "$ORIGIN_TESTS_FILTER" ${flags} --junit-dir /reports"
else
  which skopeo
  if [ $? -ne 0 ]; then
    echo "skopeo not available, exiting"
    exit 1
  fi

  if [ -f _cache/tools/openshift-tests ]; then
      echo "openshift-tests binary already present"
  else
      get_openshift_tests_binary $ORIGIN_TESTS_IMAGE ${skopeo_workdir}
  fi

  kubectl version
  EXEC_TESTS="_cache/tools/openshift-tests run "$ORIGIN_TESTS_FILTER" ${flags} --junit-dir $ORIGIN_TESTS_REPORTS_PATH"
fi

if ! eval $EXEC_TESTS; then
  failed=true
fi

if $failed; then
  echo "[WARN] Tests failed"
  exit 1
fi
