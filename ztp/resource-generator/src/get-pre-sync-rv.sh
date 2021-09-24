#!/bin/bash

for i in {0..25}
do
  . $(dirname "$0")/common.sh $1

  CM=$(oc get configmap/rv &> /dev/null)
  if (( $? == 0 )); then
    RV=$( oc get configmap/rv -ojson | jq -rM '.data.sitesResourceVersion' )
    echo "ztp-hooks.postsync $(date -R) INFO [post-sync-entrypoint] Retrieved RAN sites resourceVersion $RV" >> /proc/1/fd/2
    echo $RV
    break
  else
    echo "ztp-hooks.postsync $(date -R) ERROR [post-sync-entrypoint] Failed to get RAN sites resourceVersion" >> /proc/1/fd/2
    echo "0"
  fi
  sleep 5
done
