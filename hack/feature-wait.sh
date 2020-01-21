#!/bin/bash
set +e

if [ "$FEATURES_ENVIRONMENT" == "" ]; then
	echo "[ERROR]: No FEATURES_ENVIRONMENT provided"
	exit 1
fi

if [ "$FEATURES" == "" ]; then
	echo "[ERROR]: No FEATURES provided"
	exit 1
fi

echo "[INFO]: Sleeping before unpausing"

sleep 2m

echo "[INFO]: Unpausing"

# TODO patching to prevent https://bugzilla.redhat.com/show_bug.cgi?id=1792749 from happening
# remove this once the bug is fixed
mcps=$(oc get mcp --no-headers -o custom-columns=":metadata.name")
for mcp in $mcps
do
    oc patch mcp "${mcp}" -p '{"spec":{"paused":false}}' --type=merge
done

ELAPSED=0
TIMEOUT=120
export all_ready=false

until $all_ready || [ $ELAPSED -eq $TIMEOUT ]
do
    all_ready=true
    for feature in $FEATURES; do
      feature_ready=feature-configs/${FEATURES_ENVIRONMENT}/${feature}/is_ready.sh
      if [[ ! -f $feature_ready ]]; then    
        feature_ready=feature-configs/base/${feature}/is_ready.sh
        if [[ ! -f $feature_ready ]]; then
            continue
        fi
      fi
    
      echo "[INFO] Checking if '$feature' is ready using $feature_ready"  
      if ${feature_ready}; then
        echo "[INFO] '$feature' for environment '$FEATURES_ENVIRONMENT' is ready"
      else
        all_ready=false
      fi
    done
   sleep 10
   (( ELAPSED++ ))
done

if ! $all_ready; then 
    echo "Timed out waiting for features to be ready"
    oc get nodes
    exit 1
fi
