# Set kustomize env variable
export KUSTOMIZE_PLUGIN_HOME=$(PWD)/.config/kustomize/plugin
export XDG_CONFIG_HOME=$(KUSTOMIZE_PLUGIN_HOME)

ACM_POLICYGEN_EX_DIR=./argocd/example/acmpolicygenerator
ACM_POLICYGEN_KUSTOMIZE_DIR=$(XDG_CONFIG_HOME)/policy.open-cluster-management.io/v1/policygenerator
POLICYGEN_EX_DIR=./argocd/example/policygentemplates
POLICYGEN_KUSTOMIZE_DIR=$(XDG_CONFIG_HOME)/ran.openshift.io/v1/policygentemplate
KUSTOMIZE_DIR=/tmp
KUSTOMIZE_BIN=$(KUSTOMIZE_DIR)/kustomize
KUSTOMIZE_VERSION=5.2.1
KUSTOMIZE := $(if $(shell command -v kustomize 2>/dev/null), $(shell command -v kustomize), $(KUSTOMIZE_BIN))
POLICYGEN_DIR := ../policygenerator
SOURCE_CRS_DIR := ../source-crs
PGT2ACMPG_TOOL_DIR := ../tools/pgt2acmpg
# pgt2acmpg
ACMPG_FROM_PGT_DIR=./argocd/example/acmpgfrompgt

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
	$(PGT2ACMPG_TOOL_DIR)/scripts/build-acmpg-plugin.sh "$(ACM_POLICYGEN_KUSTOMIZE_DIR)"

$(KUSTOMIZE_BIN):
	@if [[ $(KUSTOMIZE) == $(KUSTOMIZE_BIN) ]] && [ ! -f $(KUSTOMIZE) ]; then \
		echo "kustomize not installed getting kustomize v"$(KUSTOMIZE_VERSION) \
		&& cd $(KUSTOMIZE_DIR) && curl -m 600 -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash -s $(KUSTOMIZE_VERSION); \
	fi

test: gen-files test-acmpg-from-pgt
	@echo "ZTP: Test difference between acm and ztp generated policies"
	# Run the ztp diff tool to compare the policies
	$(MAKE) -C ../tools/policy-object-template-diff ACM_PATH=../../gitops-subscriptions/$(ACM_POLICYGEN_EX_DIR)/out/ ZTP_PATH=../../gitops-subscriptions/$(POLICYGEN_EX_DIR)/out/ run


gen-files: build $(KUSTOMIZE)
	@mkdir -p $(POLICYGEN_EX_DIR)/out/
	$(KUSTOMIZE) build --enable-alpha-plugins $(POLICYGEN_EX_DIR) -o $(POLICYGEN_EX_DIR)/out/
	@mkdir -p $(ACM_POLICYGEN_EX_DIR)/out/
	$(KUSTOMIZE) build --enable-alpha-plugins $(ACM_POLICYGEN_EX_DIR) -o $(ACM_POLICYGEN_EX_DIR)/out/

clean:
	rm -rf $(ACM_POLICYGEN_KUSTOMIZE_DIR) $(POLICYGEN_KUSTOMIZE_DIR) $(POLICYGEN_EX_DIR)/out $(ACM_POLICYGEN_EX_DIR)/out $(ACM_POLICYGEN_EX_DIR)/source-crs $(POLICYGEN_EX_DIR)/source-crs $(ACMPG_FROM_PGT_DIR)

pgt2acmpg-clean:
	rm -rf ../tools/pgt2acmpg

pgt2acmpg-build:
	cd ../tools/pgt2acmpg;make build;make test

build-acmpg-from-pgt: pgt2acmpg-build
	@echo "ZTP: setup ACM from pgt kustomize plugin"
	rm -rf $(ACMPG_FROM_PGT_DIR)
	../tools/pgt2acmpg/pgt2acmpg -i $(POLICYGEN_EX_DIR) -o $(ACMPG_FROM_PGT_DIR) -s ../tools/pgt2acmpg/test/newptpconfig-schema.json -k PtpConfig,ClusterLogForwarder -g -c ../source-crs
	$(PGT2ACMPG_TOOL_DIR)/scripts/build-acmpg-plugin.sh  "$(ACM_POLICYGEN_KUSTOMIZE_DIR)"

gen-files-acmpg-from-pgt: build-acmpg-from-pgt $(KUSTOMIZE)
	@mkdir -p $(ACMPG_FROM_PGT_DIR)/out/
	$(KUSTOMIZE) build --enable-alpha-plugins $(ACMPG_FROM_PGT_DIR) -o $(ACMPG_FROM_PGT_DIR)/out/	

test-acmpg-from-pgt: gen-files gen-files-acmpg-from-pgt
	@echo "ZTP: Test difference between acm and ztp generated policies"
	# Run the ztp diff tool to compare the policies
	$(MAKE) -C ../tools/policy-object-template-diff ACM_PATH=../../gitops-subscriptions/$(ACMPG_FROM_PGT_DIR)/out/ ZTP_PATH=../../gitops-subscriptions/$(POLICYGEN_EX_DIR)/out/ run
