# Image URL to use all building/pushing image targets
# Note - you must be logged in. Do `podman login -u ... -p ... quay.io` before pushing images
# To get push permission, email to <globalcustomerservice@redhat.com>:
# Subject: Quay.io Employee access request
# - Name, Surname:
# <Your name and surname>
# - Quay.io username:
# <your quay.io username>
# - Email address that your Quay.io username is linked to (it must have @redhat domain):
# <your Red Hat email>
# - Specify the type of your request:
# Type 1
HOOK ?= quay.io/redhat_emp1/ztp-site-generator
BASE ?= quay.io/redhat_emp1/ztp-base

TAR_NAME ?= temp.tar.gz
##@ General

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

all: base build
##@ Make and push
push:	## Push to Quay.io.
	podman push ztp-site-generator:latest ${HOOK}
##@ Cluster configuration deployment
deploy: ## Deploy RBAC manifests and RAN Site CRD.
	oc apply -k ../gitops-subscriptions/argocd/deployment
undeploy: ## Undeploy RBAC manifests and RAN Site CRD.
	oc delete -k ../gitops-subscriptions/argocd/deployment
##@ Build image
base:  ## Build base image containing the CLI tools and python.
	podman build -t ztp-base:latest -f Containerfile.base && \
	podman push ztp-base:latest ${BASE}
build: ## Build the ZTP site generator image
	tar cvzf ${TAR_NAME} --exclude *.gz ../../* && \
	podman build --build-arg TAR_NAME=${TAR_NAME} -t ztp-site-generator:latest -f Containerfile; \
	rm ${TAR_NAME} && \
	podman push ztp-site-generator:latest ${HOOK}
