#!/bin/bash

. $(dirname "$0")/common.sh

# Delete old resource version configmap if present
if oc get configmap/rv &> /dev/null; then
    oc delete configmap/rv &> /dev/null
fi

RESP=$(curl -s -w "%{http_code}" $APISERVER/apis/ran.openshift.io/v1/siteconfigs --header "Authorization: Bearer $TOKEN" --cacert $CACERT)
RC=$(echo $RESP | python -c "print(input()[-3:])")
RV=$(echo $RESP | python -c "print(input()[:-3])" | jq -rM '.metadata.resourceVersion')

if [ $RC != "200" ];then
    echo "ztp-site-generator.presync $(date -R) ERROR [pre-sync-entrypoint] $APISERVER/apis/ran.openshift.io/v1/siteconfigs call returned $RC" >> /proc/1/fd/1
    exit 1
else
    # Store in configmap
    if oc create configmap rv --from-literal=sitesResourceVersion=$RV; then
        # Log even if ran manually during debugging
        echo "ztp-site-generator.presync $(date -R) INFO [pre-sync-entrypoint] Recording RAN sites resourceVersion = $RV" >> /proc/1/fd/1
    else
        echo "ztp-site-generator.presync $(date -R) ERROR [pre-sync-entrypoint] Config map creation failed" >> /proc/1/fd/1
    fi
fi
