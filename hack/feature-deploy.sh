#!/bin/bash

set -e
. $(dirname "$0")/common.sh

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

# Deploy features
success=0
iterations=0
sleep_time=10
max_iterations=72 # results in 12 minutes timeout
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
    # be verbose on last iteration only
    if [[ $iterations -eq $((max_iterations - 1)) ]] || [[ -n "${VERBOSE}" ]]; then
      # WORKAROUND for https://github.com/kubernetes/kubernetes/pull/89539:
      # oc / kubectl reject multiple manifests atm as soon as one "kind" in them does not exist yet
      # so we need to apply one manifest by one
      # since xargs' delimiter is limited to one char only or a control code, we replace the manifest delimiter "---"
      # with a "vertical tab (\v)", which should never be used in (at least our) manifests.
      # revert the sed | xargs steps when the fix landed in oc (don't forget the "else" code branch)
      ${OC_TOOL} kustomize $feature_dir | sed "s|---|\v|g" | xargs -d '\v' -I {} bash -c "echo '{}' | ${OC_TOOL} apply -f -"
      #${OC_TOOL} apply -k "$feature_dir"
    else
      ${OC_TOOL} kustomize $feature_dir | sed "s|---|\v|g" | xargs -d '\v' -I {} bash -c "echo '{}' | ${OC_TOOL} apply -f - &> /dev/null"
      #${OC_TOOL} apply -k "$feature_dir" &> /dev/null
    fi

    # shellcheck disable=SC2181
    if [[ $? != 0 ]]; then
      echo "[WARN] Deployment of feature '$feature' failed."
      feature_failed=1
    fi
    set -e

  done

  if [[ $feature_failed -eq 1 ]]; then

    iterations=$((iterations + 1))
    iterations_left=$((max_iterations - iterations))
    if [[ $iterations_left != 0  ]]; then
      echo "[WARN] Deployment did not fully succeed yet, retrying in $sleep_time sec, $iterations_left retries left"
      sleep $sleep_time
    else
      echo "[WARN] At least one deployment failed, giving up"
    fi

  else
    # All features deployed successfully
    success=1
  fi

done

if [[ $success -eq 1 ]]; then
  echo "[INFO] Deployment successful"
else
  echo "[ERROR] Deployment failed"
  exit 1
fi
