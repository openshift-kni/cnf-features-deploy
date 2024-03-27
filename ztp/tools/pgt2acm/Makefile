# Runs configured linters
lint:
	golangci-lint run --timeout 10m0s
build:
	go build -o pgt2acm
test: build
	scripts/test.sh
clean:
	rm -r test/acmgen-output
	rm pgt2acm
fetch-schema:
	kustomize openapi fetch > test/cluster-schema.json