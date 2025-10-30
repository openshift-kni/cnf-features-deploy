#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Directories that require OpenShift ZTP external kustomize plugins
# These plugins (PolicyGenerator, ClusterInstance, SiteConfig, PolicyGenTemplate) are distributed
# via Red Hat container images and are designed to run in OpenShift ArgoCD with ACM/MCE installed.
# 
# These plugins come from:
# - registry.redhat.io/openshift4/ztp-site-generate-rhel8 (ClusterInstance, SiteConfig)
# - registry.redhat.io/rhacm2/multicluster-operators-subscription-rhel9 (PolicyGenerator)
#
# They require:
# - KUSTOMIZE_PLUGIN_HOME environment variable
# - kustomize --enable-alpha-plugins flag
# - The plugin binaries extracted from the container images
#
# These can only be validated in an OpenShift cluster environment with ZTP tooling installed.
# Standard kustomize cannot process them without these prerequisites.
EXCLUDED_DIRS=(
    "./ztp/policygenerator-kustomize-plugin"
    "./ztp/siteconfig-generator-kustomize-plugin"
    "./ztp/gitops-subscriptions/argocd/example/acmpolicygenerator"
    "./ztp/gitops-subscriptions/argocd/example/policygentemplates"
    "./ztp/gitops-subscriptions/argocd/example/siteconfig"
    "./ztp/gitops-subscriptions/argocd/example/image-based-upgrades"
    "./ztp/resource-generator/telco-reference/telco-ran/configuration/argocd/example/acmpolicygenerator"
    "./ztp/resource-generator/telco-reference/telco-ran/configuration/argocd/example/policygentemplates"
    "./ztp/resource-generator/telco-reference/telco-ran/configuration/argocd/example/siteconfig"
    "./ztp/resource-generator/telco-reference/telco-ran/configuration/argocd/example/image-based-upgrades"
    "./ztp/resource-generator/telco-reference/telco-ran/configuration/argocd/example/clusterinstance"
    "./ztp/resource-generator/telco-reference/telco-ran/install/siteconfig"
    "./ztp/resource-generator/telco-reference/telco-ran/install/clusterinstance"
    "./ztp/resource-generator/telco-reference/telco-core/configuration"
    "./feature-configs/demo/local_bfd"
    "./cnf-tests/submodules/cluster-node-tuning-operator/examples/performanceprofile/default"
    "./cnf-tests/submodules/cluster-node-tuning-operator/test/e2e/performanceprofile/cluster-setup/manual-cluster/cpuFrequency"
    "./cnf-tests/submodules/metallb-operator/hack/ocp-kustomize-overlay"
    "./cnf-tests/submodules/metallb-operator/manifests/ocpcsv"
)

# Check if kustomize is installed
if ! command -v kustomize &> /dev/null; then
    echo -e "${RED}ERROR: kustomize is not installed${NC}"
    echo ""
    echo "Please install kustomize to run this check:"
    echo "  - macOS: brew install kustomize"
    echo "  - Linux: curl -s \"https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh\" | bash"
    echo "  - Manual: https://kubectl.docs.kubernetes.io/installation/kustomize/"
    echo ""
    exit 1
fi

echo "Checking all kustomization.yaml files can build successfully..."
echo ""

ERRORS=0
CHECKED=0
SKIPPED=0

# Helper function to check if directory should be excluded
is_excluded() {
    local dir="$1"
    for excluded in "${EXCLUDED_DIRS[@]}"; do
        if [ "$dir" = "$excluded" ]; then
            return 0
        fi
    done
    return 1
}

# Find all kustomization.yaml files
kustomize_files=()
while IFS= read -r file; do
    kustomize_files+=("$file")
done < <(find . -name 'kustomization.yaml' -o -name 'kustomization.yml' -not -path '*/vendor/*' -not -path '*/submodules/*' -not -path '*/.git/*' | sort)

if [ ${#kustomize_files[@]} -eq 0 ]; then
    echo -e "${YELLOW}WARNING: No kustomization.yaml files found${NC}"
    exit 0
fi

for kustomize_file in "${kustomize_files[@]}"; do
    dir=$(dirname "$kustomize_file")
    echo -n "  $dir: "
    
    # Check if this directory requires external plugins
    if is_excluded "$dir"; then
        echo -e "${BLUE}SKIPPED${NC} (requires external plugins)"
        SKIPPED=$((SKIPPED + 1))
        continue
    fi
    
    # Try to build the kustomization
    if kustomize build "$dir" > /dev/null 2>&1; then
        echo -e "${GREEN}OK${NC}"
        CHECKED=$((CHECKED + 1))
    else
        echo -e "${RED}FAILED${NC}"
        echo -e "${YELLOW}    Error details:${NC}"
        kustomize build "$dir" 2>&1 | sed 's/^/    /'
        echo ""
        ERRORS=$((ERRORS + 1))
        CHECKED=$((CHECKED + 1))
    fi
done

echo ""
echo "Summary: Checked $CHECKED kustomization.yaml files, skipped $SKIPPED (require external plugins)"

if [[ $ERRORS -eq 0 ]]; then
    echo -e "${GREEN}All kustomization files validated successfully!${NC}"
    exit 0
else
    echo -e "${RED}$ERRORS kustomization file(s) failed validation${NC}"
    exit 1
fi

