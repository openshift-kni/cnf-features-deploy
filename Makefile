#TODO add default features here
export FEATURES?=sctp performance xt_u32 vrf container-mount-namespace ovs_qos
export SKIP_TESTS?=
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
	SKIP_TESTS="$(SKIP_TESTS)" FEATURES="$(FEATURES)" hack/run-functests.sh

#validate is intended to validate the deployment as a whole, not focusing
# but eventually skipping
wait-and-validate: 
	@echo "Waiting"
	SKIP_TESTS="$(SKIP_TESTS)" DONT_FOCUS=true FEATURES="$(FEATURES)" hack/feature-wait.sh
	@echo "Validating"
	SKIP_TESTS="$(SKIP_TESTS)" DONT_FOCUS=true TEST_SUITES="validationsuite" hack/run-functests.sh

functests-on-ci: setup-test-cluster feature-deploy feature-wait functests

origin-tests:
	@echo "Running origin-tests"
	TESTS_IN_CONTAINER=true ORIGIN_TESTS_FILTER="$(ORIGIN_TESTS_FILTER)" hack/run-origin-tests.sh

skopeo-origin-tests:
	@echo "Running origin-tests"
	ORIGIN_TESTS_FILTER="$(ORIGIN_TESTS_FILTER)" hack/run-origin-tests.sh

mirror-origin-tests:
	@echo "Mirroring origin-tests"
	TESTS_IN_CONTAINER=true ORIGIN_TESTS_REPOSITORY="$(ORIGIN_TESTS_REPOSITORY)" hack/mirror-origin-tests.sh

skopeo-mirror-origin-tests:
	@echo "Mirroring origin-tests"
	ORIGIN_TESTS_REPOSITORY="$(ORIGIN_TESTS_REPOSITORY)" hack/mirror-origin-tests.sh

origin-tests-disconnected-environment:
	@echo "Mirroring origin-tests"
	ORIGIN_TESTS_REPOSITORY="$(ORIGIN_TESTS_REPOSITORY)" hack/mirror-origin-tests.sh
	@echo "Running origin-tests"
	TESTS_IN_CONTAINER=true ORIGIN_TESTS_IN_DISCONNECTED_ENVIRONMENT=true \
		ORIGIN_TESTS_REPOSITORY="$(ORIGIN_TESTS_REPOSITORY)" ORIGIN_TESTS_FILTER="$(ORIGIN_TESTS_FILTER)" \
		hack/run-origin-tests.sh

validate-on-ci: setup-test-cluster feature-deploy wait-and-validate

gofmt:
	@echo "Running gofmt"
	gofmt -s -l `find . -path ./vendor -prune -o -type f -name '*.go' -print`

golint:
	@echo "Running go lint"
	cnf-tests/hack/lint.sh

govet:
	@echo "Running go vet"
	# Disabling GO111MODULE just for go vet execution
	GO111MODULE=off go vet github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites...

ci-job: gofmt golint govet check-tests-nodesc validate-test-list

feature-deploy:
	FEATURES_ENVIRONMENT=$(FEATURES_ENVIRONMENT) FEATURES="$(FEATURES)" hack/feature-deploy.sh

setup-test-cluster:
	@echo "Setting up test cluster"
	hack/setup-test-cluster.sh

feature-wait:
	@echo "Waiting for features"
	SKIP_TESTS="$(SKIP_TESTS)" FEATURES="$(FEATURES)" hack/feature-wait.sh

test-bin:
	@echo "Making test binary"
	cnf-tests/hack/build-test-bin.sh

cnf-tests-local:
	@echo "Making cnf-tests local"
	$(IMAGE_BUILD_CMD) build --no-cache -f cnf-tests/Dockerfile -t cnf-tests-local .
	$(IMAGE_BUILD_CMD) build --no-cache -f buildingexamples/s2i-dpdk/Dockerfile -t dpdk buildingexamples/s2i-dpdk/

check-tests-nodesc:
	@echo "Checking undocumented cnf tests"
	cnf-tests/hack/fill-empty-docs.sh

generate-cnf-tests-doc:
	@echo "Generating cnf tests doc"
	cnf-tests/hack/generate-cnf-docs.sh

validate-test-list:
	@echo "Comparing newly generated docs to existing docs"
	cnf-tests/hack/compare-gen-md.sh

.PHONY: list
list:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$'
