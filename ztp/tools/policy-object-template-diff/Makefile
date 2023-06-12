GOCMD=go
BINARY_NAME=policy-object-template-diff
ACM_PATH?=""
ZTP_PATH?=""

build:
	mkdir -p out/bin
	GO111MODULE=on $(GOCMD) build -mod vendor -o out/bin/$(BINARY_NAME) .

clean:
	rm -fr ./bin
	rm -fr ./out

run:
	A_PATH=$(ACM_PATH) B_PATH=$(ZTP_PATH) out/bin/$(BINARY_NAME)
