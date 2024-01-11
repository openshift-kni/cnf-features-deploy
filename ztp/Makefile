.PHONY: ci-job test-policygen checkExtraManifests checkSourceCRsAnnotation test-policygen-kustomize test-siteconfig test-siteconfig-kustomize test-acm-ztp-generated-policies test-pkg

ci-job: test-policygen checkExtraManifests checkSourceCRsAnnotation test-policygen-kustomize test-siteconfig test-siteconfig-kustomize test-acm-ztp-generated-policies test-pkg

test-policygen:
	@echo "ZTP: Build policy generator and run test"
	$(MAKE) -C ./policygenerator all

checkExtraManifests:
	$(MAKE) -C ./extra-manifests-builder check

checkSourceCRsAnnotation:
	for sourcefile in ./source-crs/*.yaml;do \
		if [[ "$${sourcefile}" != *"MachineConfig"* ]];then \
			if ! grep -qE "ran.openshift.io/ztp-deploy-wave" "$${sourcefile}";then \
				echo "Error: missing annotation 'ran.openshift.io/ztp-deploy-wave' in $${sourcefile}"; \
				exit 1; \
			fi; \
		fi; \
	done; \

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

test-pkg:
	@echo "ZTP: test pkg"
	go test ./pkg/... -v
