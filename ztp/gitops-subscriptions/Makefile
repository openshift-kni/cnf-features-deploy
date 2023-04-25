# Set kustomize env variable
export KUSTOMIZE_PLUGIN_HOME=$(PWD)/.config/kustomize/plugin
export XDG_CONFIG_HOME=$(KUSTOMIZE_PLUGIN_HOME)

ACM_POLICYGEN_EX_DIR=./argocd/example/acmpolicygenerator
ACM_POLICYGEN_KUSTOMIZE_DIR=$(XDG_CONFIG_HOME)/policy.open-cluster-management.io/v1/policygenerator
POLICYGEN_EX_DIR=./argocd/example/policygentemplates
POLICYGEN_KUSTOMIZE_DIR=$(XDG_CONFIG_HOME)/ran.openshift.io/v1/policygentemplate
KUSTOMIZE_DIR=/tmp
KUSTOMIZE_BIN=$(KUSTOMIZE_DIR)/kustomize
KUSTOMIZE_VERSION=4.4.0
KUSTOMIZE := $(if $(shell command -v kustomize 2>/dev/null), $(shell command -v kustomize), $(KUSTOMIZE_BIN))
POLICYGEN_DIR := ../policygenerator
SOURCE_CRS_DIR := ../source-crs

.PHONY: build test gen-files clean

build:
	@echo "ZTP: Build policy generator kustomize plugin"
	$(MAKE) -C $(POLICYGEN_DIR) build
	mkdir -p $(POLICYGEN_KUSTOMIZE_DIR)
	cp -r $(SOURCE_CRS_DIR) $(POLICYGEN_EX_DIR)/
	cp $(POLICYGEN_DIR)/policygenerator $(POLICYGEN_KUSTOMIZE_DIR)/PolicyGenTemplate
	@echo "ZTP: setup ACM policyGenerator kustomize plugin"
	mkdir -p $(ACM_POLICYGEN_KUSTOMIZE_DIR)
	cp -r $(SOURCE_CRS_DIR) $(ACM_POLICYGEN_EX_DIR)
	GOBIN=$(ACM_POLICYGEN_KUSTOMIZE_DIR) go install open-cluster-management.io/policy-generator-plugin/cmd/PolicyGenerator@latest

$(KUSTOMIZE_BIN):
	@if [[ $(KUSTOMIZE) == $(KUSTOMIZE_BIN) ]] && [ ! -f $(KUSTOMIZE) ]; then \
		echo "kustomize not installed getting kustomize v"$(KUSTOMIZE_VERSION) \
		&& cd $(KUSTOMIZE_DIR) && curl -m 600 -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash -s $(KUSTOMIZE_VERSION); \
	fi

test: gen-files
	@echo "ZTP: Test difference between acm and ztp generated policies"
	# Run the ztp diff tool to compare the policies
	$(MAKE) -C ../tools/policy-object-template-diff ACM_PATH=../../gitops-subscriptions/$(ACM_POLICYGEN_EX_DIR)/out/ ZTP_PATH=../../gitops-subscriptions/$(POLICYGEN_EX_DIR)/out/ run


gen-files: build $(KUSTOMIZE)
	@mkdir -p $(POLICYGEN_EX_DIR)/out/
	$(KUSTOMIZE) build --enable-alpha-plugins $(POLICYGEN_EX_DIR) -o $(POLICYGEN_EX_DIR)/out/
	@mkdir -p $(ACM_POLICYGEN_EX_DIR)/out/
	$(KUSTOMIZE) build --enable-alpha-plugins $(ACM_POLICYGEN_EX_DIR) -o $(ACM_POLICYGEN_EX_DIR)/out/

clean:
	rm -rf $(ACM_POLICYGEN_KUSTOMIZE_DIR) $(POLICYGEN_KUSTOMIZE_DIR) $(POLICYGEN_EX_DIR)/out $(ACM_POLICYGEN_EX_DIR)/out $(ACM_POLICYGEN_EX_DIR)/source-crs $(POLICYGEN_EX_DIR)/source-crs
