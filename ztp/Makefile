.PHONY: ci-job test-policygen test-policygen-kustomize test-siteconfig test-siteconfig-kustomize test-acm-ztp-generated-policies

ci-job: init-telco-reference-submodule test-policygen test-policygen-kustomize test-siteconfig test-siteconfig-kustomize test-acm-ztp-generated-policies

init-telco-reference-submodule:
	@echo "Initializing telco-reference submodule"
	git submodule update --init --force resource-generator/telco-reference

test-policygen:
	@echo "ZTP: Build policy generator and run test"
	$(MAKE) -C ./policygenerator all

test-policygen-kustomize:
	@echo "ZTP: Build policy generator kustomize plugin and run test"
	$(MAKE) -C ./policygenerator-kustomize-plugin test

test-siteconfig:
	@echo "ZTP: Build siteconfig generator and run test"
	$(MAKE) -C ./siteconfig-generator test

test-siteconfig-kustomize:
	@echo "ZTP: Build siteconfig generator kustomize plugin and run test"
	$(MAKE) -C ./siteconfig-generator-kustomize-plugin test

test-acm-ztp-generated-policies:
	$(MAKE) -C ./gitops-subscriptions test
