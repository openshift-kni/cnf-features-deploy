export GOFLAGS := $(if $(GOFLAGS),$(GOFLAGS),-mod=vendor)

.PHONY: fmt build test vet all clean
all: fmt vet build test


fmt:
	gofmt -l -w -s .

build:
	go build ./

test:
	go test ./... -v

vet:
	go vet ./...

clean:
	rm policygenerator
	rm -rf out/
