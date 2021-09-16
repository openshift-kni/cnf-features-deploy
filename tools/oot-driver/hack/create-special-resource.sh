#!/bin/bash
set -eu

export DRIVER_TOOLKIT_IMAGE=$1; shift
export EXTERNAL_REGISTRY=$1; shift
export SIGN_DRIVER=$1; shift
export DOWNLOAD_DRIVER=$1; shift
export KERNEL_VERSION_LIST=$1; shift
export KERNEL_SOURCE=$1; shift

rm -f "./special-resource.yaml" || true
IFS=',' read -r -a array <<< "${KERNEL_VERSION_LIST}"

FILES="./templates/special-resources/*"
for f in $FILES
do
  for index in "${!array[@]}"
  do
      export OOT_DRIVER_IMAGE_NAME=$(basename ${f/special-resource.yaml.template/container/})
      export KERNEL_VERSION=${array[index]}
      export INDEX=${index}
      envsubst < "$f" >> "./special-resource.yaml"
  done
done
