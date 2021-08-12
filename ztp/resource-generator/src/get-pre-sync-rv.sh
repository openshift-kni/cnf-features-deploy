#!/bin/bash

. $(dirname "$0")/common.sh

CM=$(oc get configmap/rv &> /dev/null)
if (( $? == 0 )); then
  RV=$( oc get configmap/rv -ojson | jq -rM '.metadata.resourceVersion' )
  echo "ztp-site-generator.postsync $(date -R) INFO [post-sync-entrypoint] Retrieved RAN sites resourceVersion $RV" >> /proc/1/fd/2
  echo $RV
else
  echo "ztp-site-generator.postsync $(date -R) ERROR [post-sync-entrypoint] Failed to get RAN sites resourceVersion" >> /proc/1/fd/2
  echo "0"
fi
