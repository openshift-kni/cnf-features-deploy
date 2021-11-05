export GOFLAGS := $(if $(GOFLAGS),$(GOFLAGS),-mod=vendor)

.PHONY: build test vet fmt all

build:
	go build ./

test:
	go test ./... -v

vet:
	go vet ./...

fmt:
	go fmt ./...


all: vet fmt build test

