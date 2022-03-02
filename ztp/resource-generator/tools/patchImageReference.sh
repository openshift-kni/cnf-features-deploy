#!/bin/bash
#
# This script finds and replaces all references to our upstream container with whatever is provided on the commandline.
#
# This can be uaed for personal builds or downstream builds to ensure the contents of the container always point at the right container image
set -e

BASEDIR=$1
REPLACEMENT_IMAGE=$2
UPSTREAM_IMAGE="quay.io/openshift-kni/ztp-site-generator:latest"

if [[ $1 == "-h" || $1 == "--help" ]]; then
    echo "Usage:"
    echo "  $(basename $0) basedir quay.io/repo/container:tag"
    exit 1
fi

if [[ ! -d $BASEDIR ]]; then
    echo "FATAL: $BASEDIR is not a directory" >&2
    exit 2
fi

if [[ -z $REPLACEMENT_IMAGE || $UPSTREAM_IMAGE == $REPLACEMENT_IMAGE ]]; then
    echo "Not replacing $UPSTREAM_IMAGE"
    exit 0
fi

echo "Replacing $UPSTREAM_IMAGE with $REPLACEMENT_IMAGE..." >&2

for file in $(grep -Rl $UPSTREAM_IMAGE $BASEDIR); do
    echo "  Editing $file" >&2
    sed -i "s,$UPSTREAM_IMAGE,$REPLACEMENT_IMAGE,g" $file
done

echo "Done" >&2
exit 0
