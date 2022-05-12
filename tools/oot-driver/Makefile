INTERNAL_REGISTRY ?= default-route-openshift-image-registry.apps.test.cluster.com
AUTH_FILE ?= ""
OPENSHIFT_SECRET_FILE ?= "/root/openshift_pull.json"
DRIVER_TOOLKIT_IMAGE ?= $(shell oc adm release info --image-for=driver-toolkit -a $(OPENSHIFT_SECRET_FILE))
KERNEL_VERSION ?= $(shell hack/get_kernel_version.sh $(DRIVER_TOOLKIT_IMAGE) $(OPENSHIFT_SECRET_FILE))
KERNEL_SOURCE ?= yum
EXTERNAL_REGISTRY ?= quay.io/openshift-kni
USE_DOCKER_IMAGE ?= false
ICE_DRIVER_VERSION ?= "1.7.16"
IAVF_DRIVER_VERSION ?= "4.3.19"

SIGN_DRIVER ?= false
DOWNLOAD_DRIVER ?= false
NODE_LABEL ?= "machineconfiguration.openshift.io/role: worker-cnf"

TLS_VERIFY ?= false
HELM_REPO = $(shell ls -d charts/)

helm-lint: helm
	cd charts/; \
	for repo in `ls -d */`; do \
		helm lint -f ./global-values.yaml $$repo; \
	done

helm-repo-index: helm-lint
	cd charts/; \
	for repo in `ls -d */`; do \
		helm package $$repo; \
		helm repo index . --url=cm://oot-driver/charts; \
	done

helm-create-config-map: helm-repo-index
	charts=""
	@for repo in `cd charts && ls -d */`; do  \
  		charts=$$charts" --from-file=charts/$${repo%?}.tgz" ; \
	done; \
	oc -n oot-driver delete cm charts >/dev/null; \
	oc -n oot-driver create cm charts --from-file=charts/index.yaml $$charts


build-source-container:
	podman build -t $(INTERNAL_REGISTRY)/oot-source-driver:latest -f Dockerfile.source .

push-source-container:
	podman push --authfile=$(AUTH_FILE) --tls-verify=$(TLS_VERIFY) $(INTERNAL_REGISTRY)/oot-source-driver:latest

create-machine-config:
	./hack/create-mc.sh $(EXTERNAL_REGISTRY) $(NODE_LABEL)

apply-machine-config:
	oc apply -f oot-driver-machine-config.yaml

create-special-resource-cr:
	./hack/create-special-resource.sh $(DRIVER_TOOLKIT_IMAGE) $(EXTERNAL_REGISTRY) $(INTERNAL_REGISTRY) $(SIGN_DRIVER) \
	$(DOWNLOAD_DRIVER) "$(KERNEL_VERSION)" $(KERNEL_SOURCE) $(USE_DOCKER_IMAGE) $(ICE_DRIVER_VERSION) $(IAVF_DRIVER_VERSION)

apply-special-resource-cr:
	oc apply -f special-resource.yaml

helm:
ifeq (, $(shell which helm))
	@{ \
	set -e ;\
	HELM_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$HELM_GEN_TMP_DIR ;\
	curl https://get.helm.sh/helm-v3.6.0-linux-amd64.tar.gz -o helm.tar.gz ;\
	tar xvfpz helm.tar.gz ;\
	mv linux-amd64/helm /usr/local/bin ;\
	chmod +x /usr/local/bin/helm ;\
	rm -rf $$HELM_GEN_TMP_DIR ;\
	}
HELM=/usr/local/bin/helm
else
KUSTOMIZE=$(shell which helm)
endif
