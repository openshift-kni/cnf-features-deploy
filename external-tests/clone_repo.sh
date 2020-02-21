#!/bin/bash
set -e

if [ "$TESTS_REPO" == "" ]; then
    echo "[ERROR]: No TESTS_REPO provided"
    exit 1
fi

if [ "$TESTS_LOCATION" == "" ]; then
    echo "[ERROR]: No TESTS_LOCATION provided"
    exit 1
fi

REMOTE_BRANCH="${REMOTE_BRANCH:-master}"

# detect latest master hash of the target repo if we're not pinned to a specific commit hash
if [ -z "$TESTS_TARGET_HASH" ]; then
    echo "Using latest $REMOTE_BRANCH branch commit for $TESTS_REPO"
    TESTS_TARGET_HASH=$(git ls-remote "$TESTS_REPO" | grep refs/heads/"$REMOTE_BRANCH"$ | cut -f 1)
fi

echo "$TESTS_REPO commit hash: $TESTS_TARGET_HASH"

if ! [ -d "$TESTS_LOCATION" ]; then
    DOWNLOAD_SRC=true
elif ! [ "$(cat "$TESTS_LOCATION"/git-hash)" = "$TESTS_TARGET_HASH" ]; then
    rm -rf TESTS_LOCATION
    DOWNLOAD_SRC=true
fi

if [ $DOWNLOAD_SRC ]; then
    echo "Cloning code from $TESTS_REPO using hash $TESTS_TARGET_HASH"
    # shellcheck disable=SC2086
    repo_name=$(basename $TESTS_REPO)
    curl -L "$TESTS_REPO"/archive/"$TESTS_TARGET_HASH"/"$repo_name".tar.gz | tar xz "$repo_name"-"$TESTS_TARGET_HASH"
    mv "$repo_name"-"$TESTS_TARGET_HASH" "$TESTS_LOCATION"
else
    echo "Using cached $TESTS_LOCATION"
fi

echo "$TESTS_TARGET_HASH" > "$TESTS_LOCATION"/git-hash

