GO_PACKAGES=$(shell go list ./... | grep -v vendor)
.PHONY: all clean test
.PHONY: \
	lint \
	build \
	fetch-schema \
	vet
# Runs configured linters
lint:
	golangci-lint run --timeout 10m0s
build:
	go build -o pgt2acmpg
build-linux-amd64: build-plugins-linux-amd64
	GOOS=linux GOARCH=amd64 go build -o pgt2acmpg
build-darwin-arm64: build-plugins-darwin-arm64
	GOOS=darwin GOARCH=arm64 go build -o pgt2acmpg
clean:
	rm -r test/acmpg-output || true
	rm pgt2acmpg || true
	rm -rf build || true
	rm -rf kustomize
fetch-schema:
	kustomize openapi fetch > test/cluster-schema.json
vet:
	go vet ${GO_PACKAGES}
build-plugins-darwin-arm64: clean
	GOOS=darwin GOARCH=arm64 scripts/build-plugins.sh
build-plugins-linux-amd64: clean
	GOOS=linux GOARCH=amd64 scripts/build-plugins.sh
build-plugins:
	scripts/build-plugins.sh
build-pgt-plugin:
	scripts/build-pgt-plugin-only.sh
test:
	go test
