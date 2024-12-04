#!/bin/bash
set -o errexit -o nounset -o pipefail

SPOKE_KUBECONFIG_PATH=/etc/kubernetes/static-pod-resources/kube-apiserver-certs/secrets/node-kubeconfigs/lb-int.kubeconfig
HUB_SECRET_NAMESPACE=open-cluster-management-agent
HUB_SECRET_NAME=hub-kubeconfig-secret

# retrieves the kubeconfig for this spoke's cluster
getHubKubeconfig() {
	local kubeConfigPath namespace secretName KUBECONFIG_DATA TLS_KEY TLS_CRT

	kubeConfigPath="$1"
	namespace="$2"
	secretName="$3"
	KUBECONFIG_DATA=$(oc --kubeconfig "$kubeConfigPath" get secret -n "$namespace" "$secretName" -o json | jq .data.kubeconfig | sed 's/"//g' | base64 -d)
	if [ -z "$KUBECONFIG_DATA" ]; then
		return "$FALSE"
	fi
	TLS_KEY=$(oc --kubeconfig "$kubeConfigPath" get secret -n "$namespace" "$secretName" -o json | jq '.data."tls.key"' | sed 's/"//g')
	TLS_CRT=$(oc --kubeconfig "$kubeConfigPath" get secret -n "$namespace" "$secretName" -o json | jq '.data."tls.crt"' | sed 's/"//g')
	echo "$KUBECONFIG_DATA" | sed -e "s/client-certificate: tls.crt/client-certificate-data: $TLS_CRT/g" | sed -e "s/client-key: tls.key/client-key-data: $TLS_KEY/g" >/tmp/kubeconfig-hub
	return "$TRUE"
}

# Retreives TALM's state in the hub cluster's managedCluster object. Takes one argument:
# done -> return $TRUE if the ztp-done label is set, $FALSE otherwise
# running -> return $TRUE if the ztp-running label is set, $FALSE otherwise
isZtpState() {
	local talmState RESULT

	talmState="$1"
	RESULT=$FALSE

	clusterName=$(oc --kubeconfig "$SPOKE_KUBECONFIG_PATH" get klusterlet klusterlet -ojsonpath='{.spec.clusterName}')
	case "$talmState" in
	"running")
		RESULT=$(KUBECONFIG=/tmp/kubeconfig-hub oc get managedcluster "$clusterName" -ojson | jq '.metadata.labels["ztp-running"]!=null')
		;;
	"done")
		RESULT=$(KUBECONFIG=/tmp/kubeconfig-hub oc get managedcluster "$clusterName" -ojson | jq '.metadata.labels["ztp-done"]!=null')
		;;
	*)
		# Code to execute when no patterns match
		;;
	esac
	if [ "$RESULT" == "false" ]; then
		logDebug "TALM $talmState state is $RESULT"
		return "$FALSE"
	fi
	logDebug "TALM $talmState state is $RESULT"
	return "$TRUE"
}

isTALMUpdating() {
	if ! getHubKubeconfig $SPOKE_KUBECONFIG_PATH $HUB_SECRET_NAMESPACE $HUB_SECRET_NAME; then
		logInfo "TALM not available or hub kubeconfig is no ready yet at $SPOKE_KUBECONFIG_PATH path, cannot get spoke secret $HUB_SECRET_NAME in $HUB_SECRET_NAMESPACE namespace"
		return "$FALSE"
	fi
	isZtpState "running"
	return $?
}

# Add a new function to the array of update detection methods
serverUpdateDetectionMethods+=("isTALMUpdating")
