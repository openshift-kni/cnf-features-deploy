#!/bin/bash
set -eu

export DRIVER_TOOLKIT_IMAGE=$1; shift
export EXTERNAL_REGISTRY=$1; shift
export INTERNAL_REGISTRY=$1; shift
export SIGN_DRIVER=$1; shift
export DOWNLOAD_DRIVER=$1; shift
export KERNEL_VERSION_LIST=$1; shift
export KERNEL_SOURCE=$1; shift
export USE_DOCKER_IMAGE=$1; shift
export ICE_DRIVER_VERSION=$1; shift
export IAVF_DRIVER_VERSION=$1; shift

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
      if [ $USE_DOCKER_IMAGE = "true" ]
      then
        sed "s|name: \"oot-source-driver:latest\"|name: \"${INTERNAL_REGISTRY}\/oot-source-driver:latest\"\n          pullsecret: external-registry|g" -i ./special-resource.yaml
        sed 's/ImageStreamTag/DockerImage/g' -i ./special-resource.yaml
      fi
  done
done
