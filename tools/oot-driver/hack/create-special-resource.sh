#!/bin/bash
set -eu

export DRIVER_TOOLKIT_IMAGE=$1; shift
export EXTERNAL_REGISTRY=$1; shift
export OOT_DRIVER_IMAGE_NAME=$1; shift
export SIGN_DRIVER=$1; shift
export DOWNLOAD_DRIVER=$1; shift

envsubst < "./templates/special-resource.yaml.template" > "./special-resource.yaml"
