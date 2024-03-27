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
# pgt2acm
ACM_POLICYGEN_FROM_PGT_DIR=./argocd/example/acmpolicyfrompgt
PGT2ACM_VERSION := $(shell cat ../../version.json | jq .pgt2acm)


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
	GOBIN=$(ACM_POLICYGEN_KUSTOMIZE_DIR) go install open-cluster-management.io/policy-generator-plugin/cmd/PolicyGenerator@v1.12.4

$(KUSTOMIZE_BIN):
	@if [[ $(KUSTOMIZE) == $(KUSTOMIZE_BIN) ]] && [ ! -f $(KUSTOMIZE) ]; then \
		echo "kustomize not installed getting kustomize v"$(KUSTOMIZE_VERSION) \
		&& cd $(KUSTOMIZE_DIR) && curl -m 600 -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash -s $(KUSTOMIZE_VERSION); \
	fi

test: gen-files test-acmgen-from-pgt
	@echo "ZTP: Test difference between acm and ztp generated policies"
	# Run the ztp diff tool to compare the policies
	$(MAKE) -C ../tools/policy-object-template-diff ACM_PATH=../../gitops-subscriptions/$(ACM_POLICYGEN_EX_DIR)/out/ ZTP_PATH=../../gitops-subscriptions/$(POLICYGEN_EX_DIR)/out/ run


gen-files: build $(KUSTOMIZE)
	@mkdir -p $(POLICYGEN_EX_DIR)/out/
	$(KUSTOMIZE) build --enable-alpha-plugins $(POLICYGEN_EX_DIR) -o $(POLICYGEN_EX_DIR)/out/
	@mkdir -p $(ACM_POLICYGEN_EX_DIR)/out/
	$(KUSTOMIZE) build --enable-alpha-plugins $(ACM_POLICYGEN_EX_DIR) -o $(ACM_POLICYGEN_EX_DIR)/out/

clean:
	rm -rf $(ACM_POLICYGEN_KUSTOMIZE_DIR) $(POLICYGEN_KUSTOMIZE_DIR) $(POLICYGEN_EX_DIR)/out $(ACM_POLICYGEN_EX_DIR)/out $(ACM_POLICYGEN_EX_DIR)/source-crs $(POLICYGEN_EX_DIR)/source-crs $(ACM_POLICYGEN_FROM_PGT_DIR)

pgt2acm-clean:
	rm -rf ../tools/pgt2acm

pgt2acm-get: pgt2acm-clean
	git clone --depth 1 https://github.com/test-network-function/pgt2acm -b $(PGT2ACM_VERSION)  ../tools/pgt2acm
	cd ../tools/pgt2acm;rm -rf .git

pgt2acm-diff-check:
	rm -f ../tools/pgt2acm/pgt2acm
	rm -rf ../tools/.pgt2acm-check
	git clone --depth 1 https://github.com/test-network-function/pgt2acm -b $(PGT2ACM_VERSION)  ../tools/.pgt2acm-check
	cd ../tools/.pgt2acm-check;rm -rf .git
	cd ../tools;diff .pgt2acm-check pgt2acm

pgt2acm-build: pgt2acm-get
	cd ../tools/pgt2acm;make build

build-acmgen-from-pgt: pgt2acm-build
	@echo "ZTP: setup ACM from pgt kustomize plugin"
	rm -rf $(ACM_POLICYGEN_FROM_PGT_DIR)
	mkdir -p $(ACM_POLICYGEN_FROM_PGT_DIR)/hub-side-templating
	cp -r $(SOURCE_CRS_DIR) $(ACM_POLICYGEN_FROM_PGT_DIR)/.
	cp $(ACM_POLICYGEN_EX_DIR)/kustomization.yaml $(ACM_POLICYGEN_FROM_PGT_DIR)/
	cp $(ACM_POLICYGEN_EX_DIR)/ns.yaml $(ACM_POLICYGEN_FROM_PGT_DIR)/
	cp -r $(ACM_POLICYGEN_EX_DIR)/hub-side-templating/configMaps $(ACM_POLICYGEN_FROM_PGT_DIR)/hub-side-templating/
	../tools/pgt2acm/pgt2acm -i $(POLICYGEN_EX_DIR) -o $(ACM_POLICYGEN_FROM_PGT_DIR) -s ../tools/pgt2acm/test/schema.json -k PtpConfig -g -p
	GOBIN=$(ACM_POLICYGEN_KUSTOMIZE_DIR) go install open-cluster-management.io/policy-generator-plugin/cmd/PolicyGenerator@v1.12.4

gen-files-acmgen-from-pgt: build-acmgen-from-pgt $(KUSTOMIZE)
	@mkdir -p $(ACM_POLICYGEN_FROM_PGT_DIR)/out/
	$(KUSTOMIZE) build --enable-alpha-plugins $(ACM_POLICYGEN_FROM_PGT_DIR) -o $(ACM_POLICYGEN_FROM_PGT_DIR)/out/	

test-acmgen-from-pgt: gen-files gen-files-acmgen-from-pgt
	@echo "ZTP: Test difference between acm and ztp generated policies"
	# Run the ztp diff tool to compare the policies
	$(MAKE) -C ../tools/policy-object-template-diff ACM_PATH=../../gitops-subscriptions/$(ACM_POLICYGEN_FROM_PGT_DIR)/out/ ZTP_PATH=../../gitops-subscriptions/$(POLICYGEN_EX_DIR)/out/ run
