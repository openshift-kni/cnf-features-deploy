.PHONY: ci-job build-policygen checkSourceCrs build-policygen-kustomize

ci-job: test-policygen checkSourceCrs test-policygen-kustomize

test-policygen:
	@echo "ZTP: Build policy generator and run test"
	$(MAKE) -C ./policygenerator all

checkSourceCrs:
	$(MAKE) -C ./source-crs-builder check

test-policygen-kustomize:
	@echo "ZTP: Build policy generator kustomize plugin and run test"
	$(MAKE) -C ./policygenerator-kustomize-plugin test
