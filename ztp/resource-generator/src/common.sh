#!/bin/bash

# Setup environment, validate access to API server
# init <resourceName>
init() {
    export CHECK_RC=1
    for idx in {25..1} ; do
        export APISERVER=https://kubernetes.default.svc:443
        export TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
        export CACERT=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        export NAMESPACE=$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace)

        # Simply validate access. oc will use credentials from /var/run/secrets
        oc get configmap &> /dev/null
        if [[ $? -ne 0 ]] ; then
            echo "API check failed, $idx retries remain" >> /proc/1/fd/2
            sleep 5
        else
            export CHECK_RC=0
            break
        fi
    done

    # If we cannot access the API bail out
    if [[ $CHECK_RC -ne 0 ]] ; then
        echo "API login failed, aborting" >> /proc/1/fd/2
        exit 1
    fi

    # Provide the resource name to watch as argv[1]. If not provided,
    # will watch siteconfigs (for b/w compatability)
    export RESOURCE_NAME="siteconfigs"
    if [[ -n $1 ]]; then
        export RESOURCE_NAME=$1
    fi
}
