#!/bin/bash

source "$(dirname "$0")/../setup.sh"

which skopeo
if [ $? -ne 0 ]; then
	echo "skopeo not available, exiting"
    exit 1
fi

OSETESTS_IMAGE="${OSETESTS_IMAGE:-quay.io/openshift/origin-tests:4.8}"

function get_ose_tests_binary {
    # As CI runs in a pod, we can't directly use the image.
    # For this reason, we download the image and fetch the binary from the right layer
    skopeo copy docker://"$OSETESTS_IMAGE" oci:osetests 
    echo "Fetching openshift-tests binary from $OSETESTS_IMAGE"
    for layer in osetests/blobs/sha256/*; do
        testsbin=$(tar -t -f "$layer" | grep openshift-tests)
        if [[ $testsbin ]]; then 
            echo "Found $testsbin on $layer"
            tar xfv "$layer" "$testsbin"
            mv "$testsbin" _cache/tools/openshift-tests
            chmod +x _cache/tools/openshift-tests
            rm -rf osetests
            break
        fi
    done
}


if [ -f _cache/tools/openshift-tests ]; then 
    echo "openshift-tests binary already present"
else
    get_ose_tests_binary
fi

echo "Provider: $TEST_PROVIDER"
kubectl version
_cache/tools/openshift-tests run openshift/conformance/parallel --provider "${TEST_PROVIDER:-}" -o /tmp/artifacts/e2e.log --junit-dir /tmp/artifacts/junit
