# Set kustomize env variable
export XDG_CONFIG_HOME=./

POLICYGEN_KUSTOMIZE_DIR=./kustomize/plugin/ran.openshift.io/v1/policygentemplate
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
	cp -r $(SOURCE_CRS_DIR) $(POLICYGEN_KUSTOMIZE_DIR)/
	cp $(POLICYGEN_DIR)/policygenerator $(POLICYGEN_KUSTOMIZE_DIR)/PolicyGenTemplate

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
	rm -rf kustomize out
