.PHONY: ci-job test-policygen test-policygen-kustomize test-siteconfig test-acm-ztp-generated-policies \
	check-targets checkExtraManifests checkSourceCRsAnnotation checkZtpPaths checkPolicyPaths check-reference

TELCO_RAN_CONFIG := ./resource-generator/telco-reference/telco-ran/configuration

# Targets executed by ci-job. Keep in sync with ci-job below: check-targets fails
# the job if any of these has no recipe.
CI_TARGETS := test-policygen test-policygen-kustomize test-siteconfig test-acm-ztp-generated-policies \
	checkExtraManifests checkSourceCRsAnnotation checkZtpPaths checkPolicyPaths

ci-job: check-targets $(CI_TARGETS)

# A .PHONY target without a recipe makes `make` print "Nothing to be done" and
# exit 0, which turns CI green without checking anything. Fail instead.
check-targets:
	@failed=0; \
	for target in $(CI_TARGETS); do \
		if $(MAKE) -n "$$target" 2>&1 | grep -q "Nothing to be done"; then \
			echo "ERROR: $$target is part of ci-job but has no recipe"; \
			failed=1; \
		fi; \
	done; \
	exit $$failed

test-policygen:
	@echo "ZTP: Build policy generator and run test"
	$(MAKE) -C ./policygenerator all

test-policygen-kustomize:
	@echo "ZTP: Build policy generator kustomize plugin and run test"
	$(MAKE) -C ./policygenerator-kustomize-plugin test

test-siteconfig:
	@echo "ZTP: Build siteconfig converter and run test"
	$(MAKE) -C ./tools/siteconfig-converter test

test-acm-ztp-generated-policies:
	$(MAKE) -C ./gitops-subscriptions test

# The checks below live next to the reference CRs they validate, in the
# telco-reference submodule (run `make init-git-submodules` first).
#
# The submodule renamed this target (checkInstallExtraManifests ->
# checkExtraManifests), so use whichever one the pinned commit provides.
EXTRA_MANIFESTS_CHECK := $(shell grep -qE '^checkExtraManifests:' $(TELCO_RAN_CONFIG)/Makefile \
	&& echo checkExtraManifests || echo checkInstallExtraManifests)

checkExtraManifests:
	$(MAKE) -C $(TELCO_RAN_CONFIG) $(EXTRA_MANIFESTS_CHECK)

checkSourceCRsAnnotation:
	$(MAKE) -C $(TELCO_RAN_CONFIG) checkSourceCRsAnnotation

# Filenames must fit the ISO9660 255 byte limit with Rock Ridge extensions.
# Related issue: https://issues.redhat.com/browse/OCPBUGS-48244
checkZtpPaths:
	$(MAKE) -C $(TELCO_RAN_CONFIG) checkZtpPaths

checkPolicyPaths:
	$(MAKE) -C $(TELCO_RAN_CONFIG) checkPolicyPaths

# Kept out of ci-job on purpose: it needs kubectl-cluster_compare and
# helm-convert, which openshift-ci/Dockerfile.tools does not install.
check-reference:
	$(MAKE) -C $(TELCO_RAN_CONFIG)/kube-compare-reference check
