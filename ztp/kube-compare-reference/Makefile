GOPATH  ?= $(HOME)/go
GOBIN ?= $(GOPATH)/bin
export HELMCONVERTOR := $(GOBIN)/helm-convert
DOWNLOAD_URL := $(shell curl -s https://api.github.com/repos/openshift/kube-compare/releases/latest | jq -r '.assets[] | select(.name? | match("linux_amd64")) | .browser_download_url')
ARCHIVE_NAME := kubectl-cluster_compare-linux_amd64.tar.gz
HELM_URL := https://get.helm.sh/helm-v3.16.1-linux-amd64.tar.gz
HELM_PKG := helm-linux-amd64.tar.gz

.PHONY: check
check: metadata_lint compare

kubectl-cluster_compare:
	@echo "Downloading kube-compare tool"
	curl -sL $(DOWNLOAD_URL) -o $(ARCHIVE_NAME)
	tar zxvf $(ARCHIVE_NAME) $@
	rm -f $(ARCHIVE_NAME)

.PHONY: metadata_lint
metadata_lint: kubectl-cluster_compare
	@echo "Running kube-compare to ensure metadata.yaml is sane"
	@COMPARE_OUTPUT=$$(./kubectl-cluster_compare -r ./metadata.yaml -f /dev/null 2>&1); \
	if grep -q 'an error occurred while parsing template' <<<"$${COMPARE_OUTPUT}"; then \
		echo "Template parsing error"; \
		echo "$${COMPARE_OUTPUT}"; \
		exit 1; \
	fi; \
	echo "Okay"; \
	exit 0

.PHONY: clean
clean:
	rm -rf kubectl-cluster_compare Chartv1 renderedv1 helm


.PHONY: convert
convert: $(HELMCONVERTOR) helm
	@echo "Converting reference files to Helm Charts."
	@rm -rf Chartv1 renderedv1
	@$(HELMCONVERTOR) -r ./metadata.yaml -n Chartv1 -v default_value.yaml
	@echo "Rendering Helm Charts to CR files."
	@./helm template renderedv1 ./Chartv1 --output-dir renderedv1

.PHONY: $(HELMCONVERTOR)
$(HELMCONVERTOR):
	@echo "Installing helm-convert tool..."
	go install github.com/openshift/kube-compare/addon-tools/helm-convert@latest

helm:
	@echo "Installing helm..."
	curl -fsSL $(HELM_URL) -o $(HELM_PKG)
	tar -zxvf $(HELM_PKG) linux-amd64/helm --strip-components 1
	rm -f $(HELM_PKG)


.PHONY: compare
compare: convert
	@./compare.sh
