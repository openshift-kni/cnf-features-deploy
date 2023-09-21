REG_URL ?= quay.io/openshift-kni
IMAGE_NAME ?= ztp-site-generator
IMAGE_URL ?= $(REG_URL)/$(IMAGE_NAME)
IMAGE_REF ?=
VERSION ?= latest

.PHONY: help build export test all

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

build: ## Build the ZTP image
	cd ../..  && podman build -t ${IMAGE_URL}:${VERSION} --build-arg=IMAGE_REF='$(IMAGE_REF)' -f ztp/resource-generator/Containerfile .

export: ## Copy resources from container image to out/ directory
	mkdir -p out
	podman run --rm --log-driver=none ${IMAGE_NAME}:${VERSION} extract /home/ztp --tar | tar x -C out
	mkdir -p out/kustomize
	podman run --rm --log-driver=none ${IMAGE_NAME}:${VERSION} extract /kustomize --tar | tar x -C out/kustomize

test: ## Test entrypoints
	./tests/test-generator.sh

all: build export
