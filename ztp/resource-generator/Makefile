REG_URL ?= quay.io/redhat_emp1
IMAGE_NAME ?= ztp-site-generator
IMAGE_URL ?= $(REG_URL)/$(IMAGE_NAME)
VERSION ?= latest
TAR_NAME ?= temp.tar.gz

.PHONY: help build push export all

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

 ## Build the ztp image
build:
	tar cvzf ${TAR_NAME} --exclude *.gz ../../* && \
	podman build --build-arg TAR_NAME=${TAR_NAME} -t ${IMAGE_NAME}:${VERSION} -f Containerfile; \
	rm ${TAR_NAME}

 ## Push to registry $(REG_URL)
push:
	podman push ${IMAGE_NAME}:${VERSION} ${IMAGE_URL}:${VERSION}

## Copy resources from container image to out/ directory
export:
	mkdir -p ./out
	podman create -ti --name ztp-site-gen ${IMAGE_NAME}:${VERSION} bash
	podman cp ztp-site-gen:/kustomize ./out
	podman cp ztp-site-gen:/home/ztp ./out
	podman cp ztp-site-gen:/exportkustomize.sh ./out
	podman rm -f ztp-site-gen

all: build export push
