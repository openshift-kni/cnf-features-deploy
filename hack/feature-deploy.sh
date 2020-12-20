#!/bin/bash

export KUSTOMIZE_DIR="${KUSTOMIZE_DIR:-/tmp}"
export KUSTOMIZE_BIN=$KUSTOMIZE_DIR/kustomize
export KUSTOMIZE_VERSION=3.8.8

# expect oc to be in PATH by default
export OC_TOOL="${OC_TOOL:-oc}"

set -e

. $(dirname "$0")/common.sh


# Function: download the specified version of the Kustomize binary to the $KUSTOMIZE_DIR
function get_kustomize_binary (){
    (pushd $KUSTOMIZE_DIR
    curl -m 600 -s "https://raw.githubusercontent.com/\
kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"\
  | bash -s $KUSTOMIZE_VERSION
    popd)
}


if [ "$FEATURES_ENVIRONMENT" == "" ]; then
	echo "[ERROR]: No FEATURES_ENVIRONMENT provided"
	exit 1
fi

if [ "$FEATURES" == "" ]; then
	echo "[ERROR]: No FEATURES provided"
	exit 1
fi

if [ ! -f $KUSTOMIZE_BIN ]; then
  echo "Downloading the Kustomize tool"
  get_kustomize_binary
fi


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
      # ${OC_TOOL} apply -k "$feature_dir"
      # TODO: revert to the above after https://github.com/kubernetes/kubectl/issues/818 is merged:
      $KUSTOMIZE_BIN build "$feature_dir" | ${OC_TOOL} apply -f -
    else
      # ${OC_TOOL} apply -k "$feature_dir" &> /dev/null
      # TODO: revert to the above after https://github.com/kubernetes/kubectl/issues/818 is merged:
      $KUSTOMIZE_BIN build "$feature_dir" | ${OC_TOOL} apply -f - &> /dev/null
    fi

    # shellcheck disable=SC2181
    if [[ $? != 0 ]]; then
      echo "[WARN] Deployment of feature '$feature' failed."
      feature_failed=1
    else
      deploy_complete=feature-configs/${FEATURES_ENVIRONMENT}/${feature}/post_deploy.sh
      if [[ ! -f $deploy_complete ]]; then
        deploy_complete=feature-configs/deploy/${feature}/post_deploy.sh
        if [[ ! -f $deploy_complete ]]; then
          continue
        fi
      fi

      if ! $deploy_complete; then
        echo "[WARN] Deployment of feature '$feature' failed."
        feature_failed=1
      fi
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
