POLICYGEN_DIR := ./ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator

export GOFLAGS := $(if $(GOFLAGS),$(GOFLAGS),-mod=vendor)

.PHONY: ci-job govet
ci-job: govet

govet:
	@echo "ZTP: Running 'go vet'"
	go vet $(POLICYGEN_DIR)/...
