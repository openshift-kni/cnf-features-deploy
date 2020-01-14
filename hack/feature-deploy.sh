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

# expect kustomize to be in PATH by default
KUSTOMIZE="${KUSTOMIZE:-kustomize}"

# Deploy features
success=0
iterations=0
sleep_time=10
max_iterations=30 # results in 5 minute timeout
until [[ $success -eq 1 ]] || [[ $iterations -eq $max_iterations ]]
do

  feature_failed=0

  for feature in $FEATURES; do

    feature_dir=feature-configs/${FEATURES_ENVIRONMENT}/${feature}/
    if [[ ! -d $feature_dir ]]; then
      echo "[WARN] Feature '$feature' is not configured for environment '$FEATURES_ENVIRONMENT', skipping it"
      continue
    fi

    echo "[INFO] Deploying feature '$feature' for environment '$FEATURES_ENVIRONMENT'"
    set +e
    if ! ${KUSTOMIZE} build $feature_dir | ${OC_TOOL} apply -f -
    then
      echo "[WARN] Deployment of feature '$feature' failed."
      feature_failed=1
    fi
    set -e

  done

  if [[ $feature_failed -eq 1 ]]; then
    iterations=$((iterations + 1))
    iterations_left=$((max_iterations - iterations))
    echo "[WARN] At least one deployment failed, retrying in $sleep_time sec, $iterations_left retries left"
    sleep $sleep_time
    continue

  fi

  # All features deployed successfully
  success=1

done

if [[ $success -eq 1 ]]; then
  echo "[INFO] Deployment successful"
else
  echo "[ERROR] Deployment failed"
fi
