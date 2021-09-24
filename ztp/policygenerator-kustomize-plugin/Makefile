# Set kustomize env variable
export XDG_CONFIG_HOME=./

POLICYGEN_KUSTOMIZE_DIR=./kustomize/plugin/policyGenerator/v1/policygenerator
KUSTOMIZE_DIR=/tmp
KUSTOMIZE_BIN=$(KUSTOMIZE_DIR)/kustomize
KUSTOMIZE_VERSION=4.4.0
KUSTOMIZE := $(if $(shell command -v kustomize 2>/dev/null), $(shell command -v kustomize), $(KUSTOMIZE_BIN))
POLICYGEN_DIR := ../policygenerator

.PHONY: build test clean

build:
	@echo "ZTP: Build policy generator kustomize plugin"
	$(MAKE) -C $(POLICYGEN_DIR) build
	mkdir -p $(POLICYGEN_KUSTOMIZE_DIR)
	cp -r $(POLICYGEN_DIR)/resources $(POLICYGEN_KUSTOMIZE_DIR)/
	cp $(POLICYGEN_DIR)/policygenerator $(POLICYGEN_KUSTOMIZE_DIR)/PolicyGenerator
	@if [[ $(KUSTOMIZE) == $(KUSTOMIZE_BIN) ]] && [ ! -f $(KUSTOMIZE) ]; then \
		echo "kustomize not installed getting kustomize v"$(KUSTOMIZE_VERSION) \
		&& cd $(KUSTOMIZE_DIR) && curl -m 600 -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash -s $(KUSTOMIZE_VERSION); \
	fi

test: build
	$(KUSTOMIZE) build --enable-alpha-plugins ./

clean:
	rm $(POLICYGEN_KUSTOMIZE_DIR)/PolicyGenerator
	rm -rf $(POLICYGEN_KUSTOMIZE_DIR)/resources
	rm -rf out/

