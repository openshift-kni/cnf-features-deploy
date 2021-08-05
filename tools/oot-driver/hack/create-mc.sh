#!/bin/bash
set -eu

export EXTERNAL_REGISTRY=$1; shift
export NODE_LABEL=$1; shift
export SCRIPT=`base64 -w 0 ./files/script/oot-driver`

envsubst <  ./templates/oot-driver-machine-config.yaml.template > "./oot-driver-machine-config.yaml"

FILES="./templates/special-resources/*"
pushd ./templates/special-resources
  for i in *.template; do # Whitespace-safe but not recursive.
      export OOT_DRIVER_IMAGE_NAME=${i/special-resource.yaml.template/container}
      export OOT_DRIVER_NAME=${i/-special-resource.yaml.template/}
      envsubst < "../systemd.template" >> "../../oot-driver-machine-config.yaml"
  done
popd
