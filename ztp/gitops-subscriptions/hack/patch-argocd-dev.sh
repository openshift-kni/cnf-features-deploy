#!/usr/bin/env bash

# For dev use; if you will add the kustomize plugin binaries in the same repo->directory where the sitconfig.yaml exist
# run this script to add configManagementPlugins to the ../argocd/deployment/argocd-openshift-gitops-patch.json patch
# then re-patch the ArgoCD->openshift-gitops instance with the changes. Finally uncomment the ../argocd/deployment/clusters-app.yaml
# plugin field and apply changes.

patch='{
  "spec": {
    "configManagementPlugins": "- name: kustomize-with-local-plugins\n  generate:\n    command: [\"sh\", \"-c\"]\n    args: [\"XDG_CONFIG_HOME=./ kustomize build --enable-alpha-plugins\"]\n"
  }
}'

echo $patch &> /tmp/patch.json

jq -n 'reduce inputs as $i ({}; . * $i)' /tmp/patch.json ../argocd/deployment/argocd-openshift-gitops-patch.json

