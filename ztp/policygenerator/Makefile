export GOFLAGS := $(if $(GOFLAGS),$(GOFLAGS),-mod=vendor)

.PHONY: fmt build test vet all clean

fmt:
	go fmt ./...

build:
	go build ./

test:
	go test ./... -v

vet:
	go vet ./...

all: fmt vet build test

clean:
	rm policygenerator
	rm -rf out/
