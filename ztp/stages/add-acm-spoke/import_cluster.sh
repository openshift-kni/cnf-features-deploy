#!/usr/bin/env bash

CLUSTER_NAME=$1
PROFILE=$2
KUBE_CONFIG=$3
OUT_DIR=$4
HW_TYPE=$5

cat << EOF > $OUT_DIR/$CLUSTER_NAME.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: $CLUSTER_NAME-cluster
---
apiVersion: cluster.open-cluster-management.io/v1
kind: ManagedCluster
metadata:
  labels:
    cloud: auto-detect
    vendor: auto-detect
    name: $CLUSTER_NAME-cluster
    profile: $PROFILE
    hardwareType: $HW_TYPE
  name: $CLUSTER_NAME-cluster
spec:
  hubAcceptsClient: true
---
apiVersion: agent.open-cluster-management.io/v1
kind: KlusterletAddonConfig
metadata:
  name: $CLUSTER_NAME-cluster
  namespace: $CLUSTER_NAME-cluster
spec:
  clusterName: $CLUSTER_NAME-cluster
  clusterNamespace: $CLUSTER_NAME-cluster
  clusterLabels:
    cloud: auto-detect
    vendor: auto-detect
  applicationManager:
    enabled: true
  policyController:
    enabled: true
  searchCollector:
    enabled: true
  certPolicyController:
    enabled: true
  iamPolicyController:
    enabled: true
  version: 2.1.0
EOF

oc --kubeconfig=$KUBE_CONFIG apply -f $OUT_DIR/$CLUSTER_NAME.yaml
