export GOFLAGS := $(if $(GOFLAGS),$(GOFLAGS),-mod=vendor)

.PHONY: build test vet fmt all
all: vet fmt build test

build:
	go build ./

test:
	go test ./... -v

vet:
	go vet ./...

fmt:
	gofmt -l -w -s .
