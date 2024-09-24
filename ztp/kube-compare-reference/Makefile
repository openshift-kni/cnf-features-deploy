DOWNLOAD_URL := $(shell curl -s https://api.github.com/repos/openshift/kube-compare/releases/latest | jq -r '.assets[] | select(.name? | match("linux_amd64")) | .browser_download_url')
ARCHIVE_NAME := kubectl-cluster_compare-linux_amd64.tar.gz

.PHONY: check
check: metadata_lint

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
	rm -rf kubectl-cluster_compare
