
# Get the directory of the current makefile
# Trim any trailing slash from the directory path as we will add if when necessary later
PROJECT_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

# Get the root project directory
# Trim any trailing slash from the directory path as we will add if when necessary later
ROOT_PROJECT_DIR := $(shell dirname $(shell dirname $(abspath $(PROJECT_DIR))))

# RHEL9_RELEASE defines the RHEL9 release version to update the rpm lock file for the runtime
# This is automatically extracted from the Containerfile
RHEL9_RELEASE ?= $(shell awk '/^FROM registry.access.redhat.com\/ubi9\/ubi-minimal:/ {split($$2, parts, /[:|@]/); print parts[2]}' $(PROJECT_DIR)/Dockerfile)
RHEL9_RELEASE_DASHED := $(subst .,-,$(RHEL9_RELEASE))

# These should be set by the caller of the Makefile
RHEL9_ACTIVATION_KEY ?= ""
RHEL9_ORG_ID ?= ""

# The registry auth file is mounted into the container to allow for private registry pulls.
# This is automatically detected and mounted into the container if it exists on the host.
# If it does not exist, a warning is printed and the registry pulls may fail if not public.
# This can be set from the command line if the default is not correct for your environment.
REGISTRY_AUTH_FILE ?= $(shell echo $${XDG_RUNTIME_DIR:-/run/user/$$(id -u)})/containers/auth.json

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## Konflux Targets

.PHONY: sync-telco5g-konflux-submodule
sync-telco5g-konflux-submodule: ## Sync git submodule via the root Makefile
	$(MAKE) -C $(ROOT_PROJECT_DIR) sync-telco5g-konflux-submodule

.PHONY: konflux-update-rpm-lock-runtime
konflux-update-rpm-lock-runtime: sync-telco5g-konflux-submodule ## Update the rpm lock file for the runtime
	@echo "Updating rpm lock file for the runtime..."
	@echo "Creating lock-runtime directory..."
	mkdir -p $(PROJECT_DIR)/lock-runtime
	@echo "Copying Dockerfile to lock-runtime directory for rpm-lockfile-prototype..."
	cp $(PROJECT_DIR)/Dockerfile $(PROJECT_DIR)/lock-runtime/Dockerfile
	@echo "Copying rpms.in.yaml to lock-runtime directory..."
	cp $(PROJECT_DIR)/rpms.in.yaml $(PROJECT_DIR)/lock-runtime/rpms.in.yaml
	$(MAKE) -C $(ROOT_PROJECT_DIR)/telco5g-konflux/scripts/rpm-lock generate-rhel9-locks \
		LOCK_SCRIPT_TARGET_DIR=$(PROJECT_DIR)/lock-runtime \
		RHEL9_IMAGE_TO_LOCK=$$(awk '/^FROM registry.access.redhat.com\/ubi9\/ubi-minimal:/ {print $$2}' $(PROJECT_DIR)/Dockerfile) \
		REGISTRY_AUTH_FILE=$(REGISTRY_AUTH_FILE) \
		\
		RHEL9_RELEASE=$(RHEL9_RELEASE) \
		RHEL9_ACTIVATION_KEY=$(RHEL9_ACTIVATION_KEY) \
		RHEL9_ORG_ID=$(RHEL9_ORG_ID) \
		RHEL9_EXECUTION_IMAGE=registry.access.redhat.com/ubi9/ubi:$(RHEL9_RELEASE) \
		;
	@echo "Update rpms.lock.yaml with new contents..."
	cp $(PROJECT_DIR)/lock-runtime/rpms.lock.yaml $(PROJECT_DIR)/rpms.lock.yaml
	# intentionally keep lock-runtime directory for debugging purposes
	@echo "RPM lock file updated successfully."
