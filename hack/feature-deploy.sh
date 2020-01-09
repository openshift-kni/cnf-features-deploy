#!/bin/bash

set -e

# expect oc to be in PATH by default
OC_TOOL="${OC_TOOL:-oc}"

for f in $FEATURES; do
	echo "TODO add logic to implement feature '$f' for environment '$FEATURES_ENVIRONMENT'"
done

echo "ERROR: FEATURE DEPLOY SCRIPT $0 NEEDS IMPLEMENTATION"
