#!/bin/bash

set -e

if [ "$FEATURES_ENVIRONMENT" == "" ]; then
	echo "[ERROR]: No FEATURES_ENVIRONMENT provided"
	exit 1
fi

if [ "$FEATURES" == "" ]; then
	echo "[ERROR]: No FEATURES provided"
	exit 1
fi

# expect oc to be in PATH by default
export OC_TOOL="${OC_TOOL:-oc}"

# Destroy features

for feature in $FEATURES; do

  feature_dir=feature-configs/${FEATURES_ENVIRONMENT}/${feature}/
  if [[ ! -d $feature_dir ]]; then
    echo "[WARN] Feature '$feature' is not configured for environment '$FEATURES_ENVIRONMENT', skipping it"
    continue
  fi

  echo "[INFO] Deleting feature '$feature' for environment '$FEATURES_ENVIRONMENT'"
    set +e
    if ! ${OC_TOOL} delete -k $feature_dir
    then
      echo "[WARN] Deletion of feature '$feature' failed."
    fi
    set -e

  done
