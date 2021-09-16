POLICYGEN_DIR := ./ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator

export GOFLAGS := $(if $(GOFLAGS),$(GOFLAGS),-mod=vendor)

.PHONY: ci-job govet gotest testPolicygen
ci-job: govet gotest testPolicygen

govet:
	@echo "ZTP: Running 'go vet'"
	go vet $(POLICYGEN_DIR)/...

gotest:
	@echo "ZTP: Running 'go test'"
	go test $(POLICYGEN_DIR)/...

testPolicygen:
	$(MAKE) -C ./ztp-policy-generator generate-all
