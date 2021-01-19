#TODO add default features here
export FEATURES?=sctp performance xt_u32
IMAGE_BUILD_CMD ?= "docker"

# The environment represents the kustomize patches to apply when deploying the features
export FEATURES_ENVIRONMENT?=deploy

.PHONY: deps-update \
	functests \
	gofmt \
	golint \
	govet \
	ci-job \
	feature-deploy \
	cnf-tests-local \
	test-bin

TARGET_GOOS=linux
TARGET_GOARCH=amd64

CACHE_DIR="_cache"
TOOLS_DIR="$(CACHE_DIR)/tools"

$(shell mkdir -p $(TOOLS_DIR))

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
	# Disabling GO111MODULE just for go vet execution
	GO111MODULE=off go vet github.com/openshift-kni/cnf-features-deploy/functests/...

ci-job: gofmt golint govet check-tests-nodesc validate-test-list

feature-deploy:
	FEATURES_ENVIRONMENT=$(FEATURES_ENVIRONMENT) FEATURES="$(FEATURES)" hack/feature-deploy.sh

setup-test-cluster:
	@echo "Setting up test cluster"
	hack/setup-test-cluster.sh

feature-wait:
	@echo "Waiting for features"
	FEATURES="$(FEATURES)" hack/feature-wait.sh

test-bin:
	@echo "Making test binary"
	hack/build-test-bin.sh

cnf-tests-local:
	@echo "Making cnf-tests local"
	$(IMAGE_BUILD_CMD) build --no-cache -f cnf-tests/Dockerfile -t cnf-tests-local .
	$(IMAGE_BUILD_CMD) build --no-cache -f tools/s2i-dpdk/Dockerfile -t dpdk tools/s2i-dpdk/

check-tests-nodesc:
	@echo "Checking undocumented cnf tests"
	hack/fill-empty-docs.sh

generate-cnf-tests-doc:
	@echo "Generating cnf tests doc"
	hack/generate-cnf-docs.sh

validate-test-list:
	@echo "Comparing newly generated docs to existing docs"
	hack/compare-gen-md.sh
