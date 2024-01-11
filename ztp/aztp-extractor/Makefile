export GOFLAGS := $(if $(GOFLAGS),$(GOFLAGS),-mod=vendor)

.PHONY: fmt build test vet all clean
all: fmt vet build test


fmt:
	gofmt -l -w -s .

build:
	go build -o aztp-extractor ./

test:
	go test ./... -v

vet:
	go vet ./...

clean:
	rm aztp-extractor
	rm -rf out/
