#TODO add default features here
export FEATURES?=sctp performance vrf container-mount-namespace metallb tuningcni bondcni knmstate
export SKIP_TESTS?=
export FOCUS_TESTS?=
export METALLB_OPERATOR_TARGET_COMMIT?=main
export SRIOV_NETWORK_OPERATOR_TARGET_COMMIT?=main
export CLUSTER_NODE_TUNING_OPERATOR_TARGET_COMMIT?=main
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
	SKIP_TESTS="$(SKIP_TESTS)" FEATURES="$(FEATURES)" FOCUS_TESTS="$(FOCUS_TESTS)" JUNIT_TO_HTML=true hack/run-functests.sh

#validate is intended to validate the deployment as a whole, not focusing
# but eventually skipping
wait-and-validate: 
	@echo "Waiting"
	SKIP_TESTS="$(SKIP_TESTS)" DONT_FOCUS=true FEATURES="$(FEATURES)" hack/feature-wait.sh
	@echo "Validating"
	SKIP_TESTS="$(SKIP_TESTS)" DONT_FOCUS=true TEST_SUITES="validationsuite" hack/run-functests.sh

functests-on-ci: sync-git-submodules feature-deploy-on-ci functests

functests-on-ci-no-index-build: sync-git-submodules setup-test-cluster feature-deploy feature-wait functests

feature-deploy-on-ci: setup-test-cluster setup-build-index-image feature-deploy feature-wait

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

validate-on-ci: setup-test-cluster setup-build-index-image feature-deploy wait-and-validate

gofmt:
	@echo "Running gofmt"
	hack/gofmt.sh

golint:
	@echo "Running go lint"
	cnf-tests/hack/lint.sh

govet:
	@echo "Running go vet"
	go vet -mod=vendor ./cnf-tests/testsuites/...

verify-commits:
	hack/verify-commits.sh

verify-images-updated:
	hack/verify-images-updated.sh

ci-job: verify-commits verify-images-updated gofmt golint govet cnftests-unit
	
ztp-ci-job:
	$(MAKE) -C ztp ci-job

feature-deploy:
	FEATURES_ENVIRONMENT=$(FEATURES_ENVIRONMENT) FEATURES="$(FEATURES)" hack/feature-deploy.sh

setup-test-cluster:
	@echo "Setting up test cluster"
	hack/setup-test-cluster.sh

setup-build-index-image:
	@echo "Building custom index image for test cluster"
	hack/setup-build-index-image.sh

feature-wait:
	@echo "Waiting for features"
	SKIP_TESTS="$(SKIP_TESTS)" FEATURES="$(FEATURES)" hack/feature-wait.sh

custom-rpms:
	@echo "Installing rpms"
	RPMS_SRC="$(RPMS_SRC)" hack/custom_rpms.sh

test-bin: sync-git-submodules
	@echo "Making test binary"
	cnf-tests/hack/build-test-bin.sh

cnf-tests-local:
	@echo "Making cnf-tests local"
	$(IMAGE_BUILD_CMD) build --no-cache -f cnf-tests/Dockerfile -t cnf-tests-local .
	$(IMAGE_BUILD_CMD) build --no-cache -f buildingexamples/s2i-dpdk/Dockerfile -t dpdk buildingexamples/s2i-dpdk/

install-commit-hooks:
	git config core.hooksPath .githooks

update-helm-chart:
	cd tools/oot-driver && make helm-repo-index

.PHONY: sync-git-submodules
sync-git-submodules:
	@echo "Checking git submodules"
	@if [ "$(SKIP_SUBMODULE_SYNC)" != "yes" ]; then \
		echo "Syncing git submodules"; \
		git submodule sync --recursive; \
		git submodule update --init --recursive; \
	else \
		echo "Skipping submodule sync"; \
	fi

.PHONY: print-git-components
print-git-components:
	hack/print-git-components.sh

.PHONY: print-features
print-features:
	@echo "${FEATURES}"

.PHONY: list
list:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$'

cnftests-unit:
	@echo "Running cnf-tests utility function unit tests"
	go test ./cnf-tests/testsuites/pkg/...
