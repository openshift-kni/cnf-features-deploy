# Set kustomize env variable
export XDG_CONFIG_HOME=./

# KEEP THIS UP TO DATE (UNTIL IT WILL BE AUTOMATICALLY DETERMINED)
OCP_VERSION=4.16

SITECONFIG_V1_KUSTOMIZE_DIR=./kustomize/plugin/ran.openshift.io/v1/siteconfig
SITECONFIG_V2_KUSTOMIZE_DIR=./kustomize/plugin/ran.openshift.io/v2/siteconfig
GITOPS_OPERATOR_SUB_FILE=../gitops-subscriptions/argocd/deployment/openshift-gitops-operator.yaml
KUSTOMIZE_DIR=/tmp
KUSTOMIZE_VERSION_DIR=kustomize_version_dir
KUSTOMIZE_BIN=$(KUSTOMIZE_DIR)/kustomize
KUSTOMIZE_VERSION_DEFAULT=5.0.0
KUSTOMIZE_GET_VERSION_SCRIPT=../tools/get-kustomize-version/get-gitops-operator-kustomize-version.sh
KUSTOMIZE_VERSION=$(shell ./$(KUSTOMIZE_GET_VERSION_SCRIPT) v$(OCP_VERSION) $(GITOPS_OPERATOR_SUB_FILE) $(KUSTOMIZE_VERSION_DEFAULT) $(KUSTOMIZE_VERSION_DIR))
KUSTOMIZE := $(if $(shell command -v kustomize 2>/dev/null), $(shell command -v kustomize), $(KUSTOMIZE_BIN))
SITECONFIG_DIR := ../siteconfig-generator
EXTRAMANIFEST_DIR := ../source-crs/extra-manifest

.PHONY: build test gen-files clean

build:
	@echo "ZTP: Build siteconfig kustomize plugin"
	$(MAKE) -C $(SITECONFIG_DIR) build
	mkdir -p $(SITECONFIG_V1_KUSTOMIZE_DIR)
	mkdir -p $(SITECONFIG_V2_KUSTOMIZE_DIR)
	cp -R -L $(EXTRAMANIFEST_DIR) $(SITECONFIG_V1_KUSTOMIZE_DIR)/
	cp $(SITECONFIG_DIR)/siteconfig-generator $(SITECONFIG_V1_KUSTOMIZE_DIR)/SiteConfig
	cp -R -L $(EXTRAMANIFEST_DIR) $(SITECONFIG_V2_KUSTOMIZE_DIR)/
	cp $(SITECONFIG_DIR)/siteconfig-generator $(SITECONFIG_V2_KUSTOMIZE_DIR)/SiteConfig

$(KUSTOMIZE_BIN):
	@if [[ $(KUSTOMIZE) == $(KUSTOMIZE_BIN) ]] && [ ! -f $(KUSTOMIZE) ]; then \
		echo "kustomize not installed getting kustomize v"$(KUSTOMIZE_VERSION) \
		&& cd $(KUSTOMIZE_DIR) && curl -m 600 -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash -s $(KUSTOMIZE_VERSION); \
	fi

test: build $(KUSTOMIZE)
	$(KUSTOMIZE) build --enable-alpha-plugins ./

gen-files: build $(KUSTOMIZE)
	@mkdir -p out/
	$(KUSTOMIZE) build --enable-alpha-plugins ./ -o out/

clean:
	rm -rf kustomize out $(KUSTOMIZE_VERSION_DIR)

