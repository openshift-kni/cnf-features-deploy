export GOFLAGS := $(if $(GOFLAGS),$(GOFLAGS),-mod=vendor)

.PHONY: fmt build test vet all clean
all: fmt vet build test


fmt:
	gofmt -l -w -s .

build:
	go build -o accelerator ./

test:
	go test ./... -v

vet:
	go vet ./...

clean:
	rm accelerator
	rm -rf out/
