CLUSTER_COMPARE_DOWNLOAD_URL := $(shell curl -s https://api.github.com/repos/openshift/kube-compare/releases/latest | jq -r '.assets[] | select(.name? | match("kube-compare_linux_amd64")) | .browser_download_url')
HELM_CONVERT_DOWNLOAD_URL := $(shell curl -s https://api.github.com/repos/openshift/kube-compare/releases/latest | jq -r '.assets[] | select(.name? | match("kube-compare_addon_tools_linux_amd64")) | .browser_download_url')
HELM_URL := https://get.helm.sh/helm-v3.18.4-linux-amd64.tar.gz
HELM_PKG := helm-linux-amd64.tar.gz

.PHONY: check
check: metadata_lint compare

kubectl-cluster_compare:
	@echo "Downloading $@"
	curl -sL $(CLUSTER_COMPARE_DOWNLOAD_URL) -o $@.tgz
	tar zxvf $@.tgz $@
	rm -f $@.tgz

helm-convert:
	@echo "Downloading $@"
	curl -sL $(HELM_CONVERT_DOWNLOAD_URL) -o $@.tgz
	tar zxvf $@.tgz $@
	rm -f $@.tgz

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
convert: helm-convert helm
	@echo "Converting reference files to Helm Charts."
	@rm -rf Chartv1 renderedv1
	@./helm-convert -r ./metadata.yaml -n Chartv1 -v default_value.yaml
	@echo "Rendering Helm Charts to CR files."
	@./helm template renderedv1 ./Chartv1 --output-dir renderedv1

helm:
	@echo "Installing helm..."
	curl -fsSL $(HELM_URL) -o $(HELM_PKG)
	tar -zxvf $(HELM_PKG) linux-amd64/helm --strip-components 1
	rm -f $(HELM_PKG)


.PHONY: compare
compare: convert
	@./compare.sh "../source-crs" renderedv1

.PHONY: sync
sync: convert
	@./compare.sh --sync "../source-crs" renderedv1
