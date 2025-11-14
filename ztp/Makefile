.PHONY: ci-job test-policygen test-policygen-kustomize test-siteconfig test-siteconfig-kustomize test-acm-ztp-generated-policies check-reference checkZtpPaths

ci-job: test-policygen test-policygen-kustomize test-siteconfig test-siteconfig-kustomize test-acm-ztp-generated-policies check-reference checkZtpPaths

test-policygen:
	@echo "ZTP: Build policy generator and run test"
	$(MAKE) -C ./policygenerator all

test-policygen-kustomize:
	@echo "ZTP: Build policy generator kustomize plugin and run test"
	$(MAKE) -C ./policygenerator-kustomize-plugin test

test-acm-ztp-generated-policies:
	$(MAKE) -C ./gitops-subscriptions test
