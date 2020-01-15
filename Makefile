#TODO add default features here
export FEATURES?=sctp performance

# The environment represents the kustomize patches to apply when deploying the features
export FEATURES_ENVIRONMENT?=e2e-gcp

.PHONY: deps-update \
	functests \
	gofmt \
	golint \
	govet \
	ci-job \
	kustomize \
	feature-deploy

TARGET_GOOS=linux
TARGET_GOARCH=amd64

CACHE_DIR="_cache"
TOOLS_DIR="$(CACHE_DIR)/tools"

KUSTOMIZE_VERSION="v3.5.3"
KUSTOMIZE_PLATFORM ?= "linux_amd64"
KUSTOMIZE_BIN="kustomize"
KUSTOMIZE_TAR="$(KUSTOMIZE_BIN)_$(KUSTOMIZE_VERSION)_$(KUSTOMIZE_PLATFORM).tar.gz"
KUSTOMIZE="$(TOOLS_DIR)/$(KUSTOMIZE_BIN)"

# Export GO111MODULE=on to enable project to be built from within GOPATH/src
export GO111MODULE=on

deps-update:
	go mod tidy && \
	go mod vendor

functests:
	@echo "Running Functional Tests"
	FEATURES="$(FEATURES)" hack/run-functests.sh

functests-on-ci: setup-test-cluster feature-deploy feature-wait functests

gofmt:
	@echo "Running gofmt"
	gofmt -s -l `find . -path ./vendor -prune -o -type f -name '*.go' -print`

golint:
	@echo "Running go lint"
	hack/lint.sh

govet:
	@echo "Running go vet"
	go vet github.com/openshift-kni/cnf-features-deploy/...

ci-job: gofmt golint govet

kustomize:
	@if [ ! -x "$(KUSTOMIZE)" ]; then\
		echo "Downloading kustomize $(KUSTOMIZE_VERSION)";\
		mkdir -p $(TOOLS_DIR);\
		curl -JL https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/$(KUSTOMIZE_VERSION)/$(KUSTOMIZE_TAR) -o $(TOOLS_DIR)/$(KUSTOMIZE_TAR);\
		tar -xvf $(TOOLS_DIR)/$(KUSTOMIZE_TAR) -C $(TOOLS_DIR);\
		rm -rf $(TOOLS_DIR)/$(KUSTOMIZE_TAR);\
		chmod +x $(KUSTOMIZE);\
	else\
		echo "Using kustomize cached at $(KUSTOMIZE)";\
	fi

feature-deploy: kustomize
	KUSTOMIZE=$(KUSTOMIZE) FEATURES_ENVIRONMENT=$(FEATURES_ENVIRONMENT) FEATURES="$(FEATURES)" hack/feature-deploy.sh

setup-test-cluster:
	@echo "Setting up test cluster"
	hack/setup-test-cluster.sh

feature-wait:
	@echo "Waiting for features"
	FEATURES="$(FEATURES)" hack/feature-wait.sh
