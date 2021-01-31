#!/bin/bash

. $(dirname "$0")/origin-tests-common.sh

skopeo_workdir=$(mktemp -d)
mapping_file=$(mktemp)
function finish {
  rm -rf ${skopeo_workdir}
  rm -f ${mapping_file}
}
trap finish EXIT

# A container image repository to retrieve test images from.
# All the required test images should be mirrored to this repository.
if [ "$ORIGIN_TESTS_REPOSITORY" == "" ]; then
  echo "[ERROR]: No ORIGIN_TESTS_REPOSITORY provided, can't mirror images of $ORIGIN_TESTS_IMAGE"
  exit 1
fi

echo "Mirroring origin-tests"

if [ "$TESTS_IN_CONTAINER" == "true" ]; then
  ORIGIN_TESTS_CMD="$CONTAINER_MGMT_CLI run -it $ORIGIN_TESTS_IMAGE /usr/bin/openshift-tests"
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

  mkdir -p _cache/

  ORIGIN_TESTS_CMD="_cache/tools/openshift-tests"
fi

$ORIGIN_TESTS_CMD images --to-repository $ORIGIN_TESTS_REPOSITORY > ${mapping_file}

if [ ! -s "${mapping_file}" ]; then
  echo "[WARN] Failed to get images of $ORIGIN_TESTS_IMAGE"
  exit 1
fi

if [ -n "$MIRROR_ORIGIN_TESTS_PULL_SECRET" ]; then
  pull_secret="-a $MIRROR_ORIGIN_TESTS_PULL_SECRET"
fi

${OC_TOOL} image mirror -f ${mapping_file} ${pull_secret}
