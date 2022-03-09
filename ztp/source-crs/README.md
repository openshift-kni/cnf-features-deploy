## Source CRs
This folder contains a full list of CRs for ZTP RAN solution.

* ./ contains a list of configruation CRs to be deployed via the ACM policies
* ./extra-manifest contains the MachineConfig CRs to be applied during the cluster installation
* ./validatorCRs contains the validation CRs to be deployed via the ACM policies to validate some configuration

### Waves
Each configuration CR has a default `ran.openshift.io/ztp-deploy-wave` annotation that represents the deployment order of the resources wrapped in the ACM inform policies generated via [PolicyGen](../policygenerator/README.md). The source CR wave annotation is used for determining and setting the policy wave annotation and it will be removed from the built CR included in the generated policy at runtime.

In general, the common configuration CRs for all types of sites should be applied first(eg. CatalogSources, OperatorNamespaces, OperatorSubscriptions, etc), so they have the lowest waves. Then, the group configuration CRs for a set of similar clusters would be the next(eg. PTP configuration, PAO configuration, etc). The last is the configuration CRs for each individual site(eg. IP addresses, SRIOV configuration, etc).

To check the default wave numbers for the source CRs, simply run this command in this directory:
```
grep -r "ztp-deploy-wave" ./
```
