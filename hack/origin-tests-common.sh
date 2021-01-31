#!/bin/bash

. $(dirname "$0")/common.sh

function get_openshift_tests_binary {
    origin_tests_image=$1
    skopeo_workdir=$2
    # When running in a pod, we can't directly use the image.
    # For this reason, we download the image and fetch the binary from the right layer.
    skopeo copy docker://"$origin_tests_image" oci:${skopeo_workdir}
    echo "Fetching openshift-tests binary from $origin_tests_image"
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

export ORIGIN_TESTS_IMAGE="${ORIGIN_TESTS_IMAGE:-quay.io/openshift/origin-tests:$OCP_VERSION}"
