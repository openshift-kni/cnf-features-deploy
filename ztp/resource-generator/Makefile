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
REG ?= quay.io/redhat_emp1/ztp-site-generator


# The cnf-features-deploy zip file name. 
ZIP_NAME ?= temp.zip
all: prod

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

##@ Build image
build: make_zip build_image clean_zip ## Build container image
##@ Make and push
push:	## Push to Quay.io.
	podman push ztp-site-generator:latest ${REG}
##@ Cluster configuration deployment
deploy: ## Deploy RBAC manifests and RAN Site CRD.
	oc apply -k deployment
undeploy: ## Undeploy RBAC manifests and RAN Site CRD.
	oc delete -k deployment

make_zip:
	zip -r ${ZIP_NAME} ../../../cnf-features-deploy/ztp ../../../cnf-features-deployfeature-configs
build_image:
	podman build --build-arg ZIP_NAME=${ZIP_NAME} -t ztp-site-generator:latest -f Containerfile
clean_zip:
	rm ${ZIP_NAME} || true

