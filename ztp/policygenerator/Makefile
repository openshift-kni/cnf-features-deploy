export GOFLAGS := $(if $(GOFLAGS),$(GOFLAGS),-mod=vendor)

.PHONY: build test vet all clean

build:
	go build ./

test:
	go test ./...

vet:
	go vet ./...

all: vet build test

clean:
	rm policygenerator
	rm -rf out/
