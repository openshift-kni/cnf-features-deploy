
# Get the directory of the current makefile
# Trim any trailing slash from the directory path as we will add if when necessary later
PROJECT_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

# Get the root project directory
# Trim any trailing slash from the directory path as we will add if when necessary later
ROOT_PROJECT_DIR := $(patsubst %/,%,$(dir $(abspath $(PROJECT_DIR)/)))

# RHEL9_RELEASE defines the RHEL9 release version to update the rpm lock file for the runtime
# This is automatically extracted from the Containerfile
RHEL9_RELEASE ?= $(shell awk '/^FROM registry.access.redhat.com\/ubi9\/ubi-minimal:/ {split($$2, parts, /[:|@]/); print parts[2]}' $(PROJECT_DIR)/.konflux/Dockerfile)
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

.PHONY: sync-git-submodules
sync-git-submodules: ## Sync git submodules via the root Makefile
	$(MAKE) -C $(ROOT_PROJECT_DIR) sync-git-submodules

.PHONY: konflux-update-rpm-lock-runtime
konflux-update-rpm-lock-runtime: sync-git-submodules ## Update the rpm lock file for the runtime
	@echo "Updating rpm lock file for the runtime..."
	@echo "Copying Dockerfile to lock-runtime directory for rpm-lockfile-prototype..."
	cp $(PROJECT_DIR)/.konflux/Dockerfile $(PROJECT_DIR)/.konflux/lock-runtime/Dockerfile
	@echo "Resolving RHEL_VERSION variable in copied Dockerfile..."
	sed -i '' 's/$${RHEL_VERSION}/$(RHEL9_RELEASE)/g' $(PROJECT_DIR)/.konflux/lock-runtime/Dockerfile
	$(MAKE) -C $(ROOT_PROJECT_DIR)/telco5g-konflux/scripts/rpm-lock generate-rhel9-locks \
		LOCK_SCRIPT_TARGET_DIR=$(PROJECT_DIR)/.konflux/lock-runtime \
		RHEL9_IMAGE_TO_LOCK=$$(awk '/^FROM registry.access.redhat.com\/ubi9\/ubi-minimal:/ {split($$2, parts, /[:|@]/); print parts[1] ":" parts[2]}' $(PROJECT_DIR)/.konflux/Dockerfile)
		REGISTRY_AUTH_FILE=$(REGISTRY_AUTH_FILE) \
		\
		RHEL9_RELEASE=$(RHEL9_RELEASE) \
		RHEL9_ACTIVATION_KEY=$(RHEL9_ACTIVATION_KEY) \
		RHEL9_ORG_ID=$(RHEL9_ORG_ID) \
		RHEL9_EXECUTION_IMAGE=registry.access.redhat.com/ubi9/ubi:$(RHEL9_RELEASE) \
		;
	@echo "Cleaning up temporary Dockerfile..."
	rm -f $(PROJECT_DIR)/.konflux/lock-runtime/Dockerfile
	@echo "Rpm lock file updated successfully."
