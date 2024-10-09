#!/bin/bash
set -o errexit -o nounset -o pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CNF_FEATURE_DEPLOY_ROOT=$SCRIPT_DIR/../../../..

# Build latest Policy Generator Template plugin executable
cd "$CNF_FEATURE_DEPLOY_ROOT"/ztp/policygenerator 
make build
cp policygenerator "$1"/PolicyGenTemplate
