#!/bin/bash

export APISERVER=https://kubernetes.default.svc:443
export TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
export CACERT=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt
export NAMESPACE=$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace)

export LOGIN_RC=1
for idx in {25..1} ; do
    oc login $APISERVER --token=$TOKEN --certificate-authority=$CACERT &> /dev/null
    if [[ $? -eq 0 ]] ; then
        echo "API login failed, $idx retries remain" >> /proc/1/fd/2
        sleep 5
    else
        export LOGIN_RC=0
    fi
done

# If we cannot log into the API bail out
if [[ $LOGIN_RC -ne 0 ]] ; then
    echo "API login failed, aborting" >> /proc/1/fd/2
    exit 1
fi

# Provide the resource name to watch as argv[1]. If not provided,
# will watch siteconfigs (for b/w compatability)
export RESOURCE_NAME="siteconfigs"
if [[ -n $1 ]]; then
    export RESOURCE_NAME=$1
fi

