# ACM PolicyGenerator 
ACM Policy Generator Custom Resource (CRs) are the recommended alternative to Policy Generator Templates (PGT) CRs. Both CR types are used to generate ACM policies. ACM Policy Generator CRs are similar to PGT, but there are a few notable differences regarding its patching and placement strategies.

The [policy-generator-plugin](https://github.com/stolostron/policy-generator-plugin/policy-generator-plugin) examples in this directory are defining the DU profile policies using [ACM PolicyGenerator reference API](https://github.com/stolostron/policy-generator-plugin/policy-generator-plugin/blob/main/docs/policygenerator-reference.yaml). 

These examples will generate ACM policies same as the DU profile policies generated from [policygentemplates example](https://github.com/openshift-kni/cnf-features-deploy/tree/master/ztp/gitops-subscriptions/argocd/example/policygentemplates) AND specifically the policy content object-definition (source-crs) are identical.

# Comparison between PGT and ACM Policy Generator patching strategies

| PGT patch | ACMPG patch |
|-----------|---------------------|
| Uses Kustomize merge strategies [link](https://github.com/kubernetes-sigs/kustomize/blob/master/examples/patchMultipleObjects.md)|PGT patches work by replacing variables with their values as defined by the patch|
|Relies only on patching, no embedded variable substitution|Overwrites values defined in patch|
|Does not support merging lists, only replace (missing support for OpenAPI schema)|Substitute variables defined in source CR with values defined in patch (for example $name)|
|Requires additional directives ($patch: replace) in patch to merge content that does not follow a schema (PTP plugins object)| Can overwrite the name and namespace defined in source CR reference|

# Using ACM Policy Generator templates


## Editing example templates from scratch
Using ACM Policy Generator templates is similar to PGT as described in chapter 3 of the following readme describing the overall ZTP process [link](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/gitops-subscriptions/argocd/README.md)

3. Create the ACMPG Template CR for your site in your local clone of the git repository:
   1. Begin by choosing an appropriate example from out/argocd/example/acmpolicygenerator. This directory demonstrates a 3-level policy framework which represents a well-supported low-latency profile tuned for the needs of 5G Telco DU deployments:
      - A single [acm-common-ranGen.yaml](ztp/gitops-subscriptions/argocd/example/acmpolicygenerator/acm-common-ranGen.yaml) should be applied to SNO, and do not use [acm-common-mno-ranGen.yaml](ztp/gitops-subscriptions/argocd/example/acmpolicygenerator/acm-common-mno-ranGen.yaml) file for SNO clusters.
      - For MNO clusters, it will require both [acm-common-ranGen.yaml](ztp/gitops-subscriptions/argocd/example/acmpolicygenerator/acm-common-ranGen.yaml) and [acm-common-mno-ranGen.yaml](ztp/gitops-subscriptions/argocd/example/acmpolicygenerator/acm-common-mno-ranGen.yaml) file.
      - A set of shared `acm-group-du-*-ranGen.yaml`, each of which should be common across a set of similar clusters
      - An example [acm-example-sno-site.yaml](ztp/gitops-subscriptions/argocd/example/policygentemplates/acm-example-sno-site.yaml) which will normally be copied and updated for each individual site
   2. Ensure the labels defined in your PGTs `bindingRules` section correspond to the proper labels defined on the SiteConfig file(s) of the clusters you are managing.
   3. Ensure the content of the overlaid spec files matches your desired end state.  As a reference, the out/source-crs directory contains the full list of source-crs available to be included and overlayed by your PGT templates.
      - Note: Depending on the specific requirements of your clusters, you may need more than just a single group policy per cluster type, especially considering the example group policies each has a single PerformancePolicy which can only be shared across a set of clusters if those clusters consist of identical hardware configurations.
   4. Define all the policy namespaces in a yaml file much like in the example out/argocd/example/policygentemplates/ns.yaml
   5. Add all the ACM Policy Generator templates files and ns.yaml to the kustomization.yaml file, much like in the example [kustomization.yaml](ztp/gitops-subscriptions/argocd/example/acmpolicygenerator/kustomization.yaml)
   6. Commit the PolicyGenTemplate CRs, ns.yaml, and associated kustomization.yaml in git.
4. Push your changes to the git repository, and the ArgoCD pipeline will detect the changes and begin the site deployment. The SiteConfig and PolicyGenTemplate CRs can be pushed simultaneously. Note: The policyGenTemplate CRs and associated ns.yaml, kustomization.yaml must be pushed to the git repository within the 20 mins after the SiteConfigs are pushed.

## Adding ManagedClustersetbinding 
A ManagedClusterSet object brings together managed clusters with same access rights. In ZTP, the default clusterset is named `global`.
With ACM Policy Generator templates, it is required to specify a clusterset binding. The ManagedClusterSetBinding adds a namespace to to the list of namespaces allowed to managed the managed clusters in the clusterset.
The ManagedClusterSetBinding can be added to the ns.yaml file. The managed ManagedClusterSetBinding below adds the `ztp-common`, `ztp-group` and `ztp-site` namespaces to the list of namespaces part of the `global` Clusterset

```
---
apiVersion: cluster.open-cluster-management.io/v1beta2
kind: ManagedClusterSetBinding
metadata:
  name: global
  namespace: ztp-common
spec:
  clusterSet: global
---
apiVersion: cluster.open-cluster-management.io/v1beta2
kind: ManagedClusterSetBinding
metadata:
  name: global
  namespace: ztp-group
spec:
  clusterSet: global
---
apiVersion: cluster.open-cluster-management.io/v1beta2
kind: ManagedClusterSetBinding
metadata:
  name: global
  namespace: ztp-site
spec:
  clusterSet: global
```

# Patching CR objects containing lists
Creating patches for CR objects containing lists is not currently supported by ACM the Generator Plugin. As a workwaround, the full content of final object must be contained in the patch. 
The pgt2acmpg tool supports creating such a patch that contains the full content of the CR, see below.

# The ACM PolicyGenerator version of the DU reference configuration
The ACM PolicyGenerator version of this reference configuration is functionally
identical to the PolicyGenTemplate version. The following sections describe some
of the key aspects of the PolicyGenerator version.

For more general info using PolicyGenerator follow the [ACM PolicyGenerator
examples](https://github.com/stolostron/policy-generator-plugin/policy-generator-plugin/tree/main/examples).


### ArgoCD Application setup

The ACM PolicyGenerator version of the DU reference can be used with the same
ArgoCD application
[policies-app](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/gitops-subscriptions/argocd/deployment/policies-app.yaml)
that is used to deploy the DU profile policygentemplates examples. For more info
how to setup the ArgoCD policies application follow the
[ReadMe](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/gitops-subscriptions/argocd/README.md)
section "Preparation of Hub cluster for ZTP". The Git repo that will be used
with the ArgoCD policies application should contain the source-crs directory and
must co-exist with the DU profile ACM PolicyGenerator as shown below as example

```
├── acmpolicygenerator
│   ├── source-crs/
│   │    ├── AcceleratorsNS.yaml
│   │    ├── ...
├── acm-common-ranGen.yaml
├── acm-example-multinode-site.yaml
├── acm-example-sno-site.yaml
├── acm-group-du-3node-ranGen.yaml
├── acm-group-du-3node-validator-ranGen.yaml
├── acm-group-du-clo5-cleanup.yaml
├── acm-group-du-sno-ranGen.yaml
├── acm-group-du-sno-validator-ranGen.yaml
├── acm-group-du-standard-ranGen.yaml
├── acm-group-du-standard-validator-ranGen.yaml
├── kustomization.yaml
├── ns.yaml

```

### Upgrade Cluster Logging Operator to 6.0

The Cluster Logging Operator (CLO) move from version 5.y to 6.0 required
adaptation to a new API and careful management of the transition during cluster
upgrades. The Operator itself could upgrade using typical Subscription channel
changes (as rolled out by TALM), however the API change required a new CR to be
created *followed by* deletion of the old API CRs. This ordering ensures that
logs will be streamed from the cluster without interruption or massive
duplication -- logs will be duplicated while both collectors are running in
parallel but the new collectors will not restart at the beginning of the log
files.

The policy which removes the CLO 5.y API artifacts must account for two
scenarios
 - upgraded clusters where the 5.y CRDs and artifacts exist and must be removed
 - newly deployed clusters where the 5.y CRDs never existed and the types are thus unknown

To avoid having two separate policies, one for upgrade and one for newly
deployed clusters, the `acm-group-du-clo5-cleanup` policy includes
`ClusterLogging5Cleanup.yaml` which is not a true "source CR". This file is an
ACM Policy `object-template-raw` which enables us to query for existence of the
CRD and, iff it exists, remove the old API CR and the CRD. This leverages the
ACM PolicyGenerator support for source files containing object-template-raw
content which is available from ACM 2.10+.

The `acm-group-du-clo5-cleanup` PolicyGenerator was used to statically generate
the Policy CR available in the ../policygentemplates directory:
`group-du-clo5-cleanup-policy.yaml`. This ensures that the Policy applied to the
hub cluster is consistent whether the environment uses PolicyGenTemplates (and
includes the statically generated Policy) or PolicyGenerator.

### Other

The pgt2acmpg supports converting Policy Gen Templates to ACM Policy Generator templates. More details can be found at [link](https://github.com/openshift-kni/cnf-features-deploy/ztp/tools/pgt2acmpg/blob/main/README.md)
