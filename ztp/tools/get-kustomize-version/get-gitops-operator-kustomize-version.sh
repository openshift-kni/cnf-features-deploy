#!/bin/bash

# Script to determine the kustomize version used in the corresponding argocd container
# of the current openshift-gitops operator.

# Check for empty parameters.
if [[ $# -ne 4 ]]; then
  exit 1
fi

# From now on return 0 even in case of error since we default to a kustomize version.

# Save the parameters to variables with meaningful names.
# Usually the OCP version.
index_tag=$1
# ztp/gitops-subscriptions/argocd/deployment/openshift-gitops-operator.yaml
openshift_gitops_operator_sub=$2
# Default kustomize version.
default_kustomize_version=$3
# The name of the local test directory.
test_dir=$4

mkdir -p "$test_dir"
# Used files.
index_results_file="$test_dir/index_results"

cleanup() {
    rm -f "$test_dir"
}

# Check the second parameter points to a file that exists.
if [[ ! -f "$2" ]]; then
  echo "$default_kustomize_version"
  exit 0
fi

# Render file based catalog from redhat-operator-index.
if ! podman run --rm registry.redhat.io/redhat/redhat-operator-index:"$index_tag" render /configs/ > "$index_results_file"; then
    echo "$default_kustomize_version"
    exit 0
fi

# Extract the latest desired openshift-gitops channel.
if ! gitops_channel=$(awk '/channel: gitops-/ {print $2}' "$openshift_gitops_operator_sub"); then
    echo "$default_kustomize_version"
    exit 0
fi

if [[ -z $gitops_channel ]]; then
    echo "$default_kustomize_version"
    exit 0
fi

# Extract the last openshift-gitops image.
gitops_version=$(echo "$gitops_channel" | grep -oE "[0-9.\]+")
gitops_image=$(jq -r \
  --arg gitops_channel "$gitops_channel" \
  --arg gitops_version "$gitops_version" \
  'select(.name == $gitops_channel) | .entries[] | select(.name | contains($gitops_version)) | .name' \
  "$index_results_file" | \
  tail -n 1)
if [[ $? -ne 0 || -z $gitops_image ]]; then
    echo "$default_kustomize_version"
    exit 0
fi

# Extract the related argocd image.
argocd_image=$(jq -r \
  --arg gitops_image "$gitops_image" \
  'select(.name == $gitops_image).relatedImages[] | select (.name == "argocd_image") | .image' \
  "$index_results_file")
if [[ $? -ne 0 || -z $argocd_image ]]; then
    echo "$default_kustomize_version"
    exit 0
fi

# Get the kustomize version.
if ! kustomize_version=$(podman run --rm "$argocd_image" kustomize version | grep -oE "v[0-9.\]+" | grep -oE "[0-9.\]+"); then
  echo "$default_kustomize_version"
  exit 0
fi

# If not empty, print the resulted kustomize version and exit.
if [[ -n $kustomize_version ]]; then
  echo "$kustomize_version"
  exit 0
fi

# Print the default kustomize version.
echo "$default_kustomize_version"

