GO_PACKAGES=$(shell go list ./... | grep -v vendor)

.PHONY: install-lint \
	lint \
	vet

# Get default value of $GOBIN if not explicitly set
GO_PATH=$(shell go env GOPATH)
ifeq (,$(shell go env GOBIN))
  GOBIN=${GO_PATH}/bin
else
  GOBIN=$(shell go env GOBIN)
endif

# Variables
GOLANGCI_VERSION=v1.52.1

vet:
	go vet ${GO_PACKAGES}

# Install golangci-lint	
install-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${GO_PATH}/bin ${GOLANGCI_VERSION}

# Run configured linters
lint:
	golangci-lint run --timeout 10m0s
