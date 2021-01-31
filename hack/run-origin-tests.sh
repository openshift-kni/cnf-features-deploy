#!/bin/bash

. $(dirname "$0")/common.sh

skopeo_workdir=$(mktemp -d)
function finish_skopeo {
  rm -rf ${skopeo_workdir}
}
trap finish_skopeo EXIT

function get_openshift_tests_binary {
    # When running in a pod, we can't directly use the image.
    # For this reason, we download the image and fetch the binary from the right layer.
    skopeo copy docker://"$ORIGIN_TESTS_IMAGE" oci:${skopeo_workdir}
    echo "Fetching openshift-tests binary from $ORIGIN_TESTS_IMAGE"
    for layer in ${skopeo_workdir}/blobs/sha256/*; do
        echo "layer=${layer}"
        set +e
        testsbin=$(tar -t -f "$layer" | grep openshift-tests)
        set -e
        if [[ $testsbin ]]; then
            echo "Found $testsbin on $layer"
            tar xfv "$layer" "$testsbin"
            mv "$testsbin" _cache/tools/openshift-tests
            chmod +x _cache/tools/openshift-tests
            rm -rf ${skopeo_workdir}
            break
        fi
    done
}

ORIGIN_TESTS_CONTAINER_MGMT_CLI="${ORIGIN_TESTS_CONTAINER_MGMT_CLI:-docker}"
ORIGIN_TESTS_REPORTS_PATH="${ORIGIN_TESTS_REPORTS_PATH:-/tmp/artifacts/}"

ORIGIN_TESTS_IN_CONTAINER="${ORIGIN_TESTS_IN_CONTAINER:-true}"
ORIGIN_TESTS_IMAGE="${ORIGIN_TESTS_IMAGE:-quay.io/openshift/origin-tests:$OCP_VERSION}"
ORIGIN_TESTS_FILTER="${ORIGIN_TESTS_FILTER:-openshift/conformance/parallel}"
CLUSTER_PROVIDER="${CLUSTER_PROVIDER:-}"

failed=false

echo "Running $ORIGIN_TESTS_FILTER tests"

mkdir -p "$ORIGIN_TESTS_REPORTS_PATH"

mkdir -p _cache/
cp -f "$KUBECONFIG" _cache/kubeconfig

# The cluster provider is the infrastructure provider (azure, aws, etc.).
# Can be null when running on baremetal
if [ -n "$CLUSTER_PROVIDER" ]; then
  echo "Provider: $CLUSTER_PROVIDER"
  provider="--provider "${CLUSTER_PROVIDER}""
fi

if [ "$ORIGIN_TESTS_IN_CONTAINER" == "true" ]; then
  EXEC_TESTS="$ORIGIN_TESTS_CONTAINER_MGMT_CLI run -v $(pwd)/_cache/:/kubeconfig:Z -v $ORIGIN_TESTS_REPORTS_PATH:/reports:Z \
  -e KUBECONFIG=/kubeconfig/kubeconfig -it $ORIGIN_TESTS_IMAGE \
  /usr/bin/openshift-tests run "$ORIGIN_TESTS_FILTER" ${provider} --junit-dir /reports"
else
  which skopeo
  if [ $? -ne 0 ]; then
    echo "skopeo not available, exiting"
    exit 1
  fi

  if [ -f _cache/tools/openshift-tests ]; then
      echo "openshift-tests binary already present"
  else
      get_openshift_tests_binary
  fi

  kubectl version
  EXEC_TESTS="_cache/tools/openshift-tests run "$ORIGIN_TESTS_FILTER" ${provider} --junit-dir $ORIGIN_TESTS_REPORTS_PATH"
fi

if ! $EXEC_TESTS; then
  failed=true
fi

if $failed; then
  echo "[WARN] Tests failed"
  exit 1
fi
