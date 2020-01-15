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

ELAPSED=0
TIMEOUT=60
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
    exit 1
fi
