.PHONY: ci-job test-policygen checkExtraManifests checkSourceCRsAnnotation test-policygen-kustomize test-siteconfig test-siteconfig-kustomize test-acm-ztp-generated-policies check-reference checkSourceCRsPath

ci-job: test-policygen checkExtraManifests checkSourceCRsAnnotation test-policygen-kustomize test-siteconfig test-siteconfig-kustomize test-acm-ztp-generated-policies check-reference checkSourceCRsPath

test-policygen:
	@echo "ZTP: Build policy generator and run test"
	$(MAKE) -C ./policygenerator all

checkExtraManifests:
	$(MAKE) -C ./extra-manifests-builder check

checkSourceCRsAnnotation:
	for sourcefile in ./source-crs/*.yaml;do \
		if [[ "$${sourcefile}" != *"MachineConfig"* && "$${sourcefile}" != *"/ConfigMapGeneric.yaml" ]];then \
			if ! grep -qE "ran.openshift.io/ztp-deploy-wave" "$${sourcefile}";then \
				echo "Error: missing annotation 'ran.openshift.io/ztp-deploy-wave' in $${sourcefile}"; \
				exit 1; \
			fi; \
		fi; \
	done; \

source-crs := source-crs
checkSourceCRsPath:
	@failures=0; \
	for cr in $(shell find $(source-crs) -type f); do \
	  path_length=$$(echo $$cr | wc -c); \
	  if [ $$path_length -gt 255 ]; then \
	    echo "File path too long: $$cr (length: $$path_length)"; \
		(( failures += 1 )); \
	  else \
	    echo "File path OK: $$cr (length: $$path_length)"; \
	  fi; \
	done; \
	exit $$failures

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

check-reference:
	$(MAKE) -C ./kube-compare-reference check
