## Image-Based Upgrades (IBU)
This directory contains examples of generating resources required for Image Based Upgrades (IBU) utilizing the [Life Cycle Agent operator](https://github.com/openshift-kni/lifecycle-agent). These examples define policies to automate image-based upgrades, ensuring seamless deployment across managed clusters through Gitops.


### Prerequisites

* Advanced Cluster Management (ACM) 2.10+
* Before using the IBU examples, ensure that the following namespaces have been created:
  - `ztp-group`: The ibu policies will be created in this namespace. If you use another name for the `group` namespace, please remember to add the namespace in [ns.yaml](../policygentemplates/ns.yaml)
  - `openshift-adp`: The ConfigMap containing the related OpenShift API for Data Protection (OADP) Custom Resources (CRs) will be copied to this namespace on the applicable spoke cluster(s).

### Setup ArgoCD Application

To deploy the IBU examples, you can use the existing [policies-app](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/gitops-subscriptions/argocd/deployment/policies-app.yaml), which is also used for deploying DU profile policy examples. Refer to the [ReadMe](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/gitops-subscriptions/argocd/README.md) section "Preparation of Hub cluster for ZTP" for detailed instructions on setting up the ArgoCD policies application.

Ensure that your Git repository, which will be used with the ArgoCD policies application, contains a directory structured as follows:

```plaintext
├── source-crs/
│   ├── ibu/
│   │    ├── ImageBasedUpgrade.yaml
│   │    ├── PlatformBackupRestore.yaml
│   │    ├── PlatformBackupRestoreLvms.yaml
├── ...
├── custom-oadp-workload-crs.yaml
├── ibu-upgrade-ranGen.yaml
├── kustomization.yaml
```

Note that [`source-crs/ibu`](https://github.com/openshift-kni/cnf-features-deploy/tree/master/ztp/source-crs/ibu) is provided in the ZTP image, however, it is important to ensure that the `kustomization.yaml` file is located in the same directory structure shown above in order to reference the ibu manifests.

### Generating the OADP ConfigMap and Policies

To generate the OADP ConfigMap encapsulating the OADP backup and restore CRs for IBU, use the [`configMapGenerator`](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/#configmapgenerator) provided by the Kustomize tool with Platform and Application(optional) backup and restore source files defined in it.
As shown in the example below, this will create a Configmap named `oadp-cm` in the namespace `ztp-group` namespace on the hub cluster. 

```yaml
configMapGenerator:
- files:
  - source-crs/ibu/PlatformBackupRestore.yaml
  # - source-crs/ibu/PlatformBackupRestoreLvms.yaml
  # - custom-oadp-workload-crs.yaml
  name: oadp-cm
  namespace: ztp-group

generatorOptions:
  disableNameSuffixHash: true
```

* [PlatformBackupRestore.yaml](../../../../source-crs/ibu/PlatformBackupRestore.yaml) is provided to backup and restore ACM klusterlet related resources.
* [PlatformBackupRestoreLvms.yaml](../../../../source-crs/ibu/PlatformBackupRestoreLvms.yaml)(optional) is provided for use cases when the LVMS is configured in the cluster as the storage solution.
* `custom-oadp-workload-crs.yaml`(optional) defines the OADP backup and restore CRs for the additional workload running on the target cluster. Ensure that the `custom-oadp-workload-crs.yaml` file includes a one-to-one mapping of OADP backup and restore CRs. It's important to note that these CRs can be stored either in separate YAML manifests or consolidated within a single YAML file (as shown below), with each CR section separated by the `---` directive.

```yaml
apiVersion: velero.io/v1
kind: Backup
metadata:
  labels:
    velero.io/storage-location: default
  name: foobar-app
  namespace: openshift-adp
spec:
  includedNamespaces:
  - foobar
  includedNamespaceScopedResources:
  - secrets
  - deployments
  - statefulsets
  excludedClusterScopedResources:
  - persistentVolumes
---
apiVersion: velero.io/v1
kind: Restore
metadata:
  name: foobar-app
  namespace: openshift-adp
  labels:
    velero.io/storage-location: default
  annotations:
    lca.openshift.io/apply-wave: "3"
spec:
  backupName:
    foobar-app
```

Choose either [ibu-upgrade-ranGen.yaml](./ibu-upgrade-ranGen.yaml) example using ZTP `PolicyGenTemplate` or [acm-ibu-upgrade-ranGen.yaml](./acm-ibu-upgrade-ranGen.yaml) example using ACM `PolicyGenerator` to create policies for performing IBU. Both examples generate the same policies as following:
* group-ibu-oadp-cm-policy: propagate OADP configmap from hub cluster to target spoke clusters in the `openshift-adp` namespace
* group-ibu-prep-policy: to transition ibu to Prep stage
* group-ibu-upgrade-policy: to transition ibu to Upgrade stage
* group-ibu-finalize-policy: to transition ibu to Idle stage
* group-ibu-rollback-policy(optional): to transition ibu to Rollback stage

Add the template to [kustomization.yaml](./kustomization.yaml) file in the `generators` object.
```yaml
generators:
# Use policygentemplate to create oadp cm and ibu policies
- ibu-upgrade-ranGen.yaml
# Use acmpolicygenerator to create oadp cm and ibu policies
# - acm-ibu-upgrade-ranGen.yaml
```

When `ibu-upgrade-ranGen.yaml` is used, override the oadp configmap data field with hub template using the Kustomize patches.
```yaml
patches:
- target:
    group: policy.open-cluster-management.io
    version: v1
    kind: Policy
    name: group-ibu-oadp-cm-policy
  patch: |-
    - op: replace
      path: /spec/policy-templates/0/objectDefinition/spec/object-templates/0/objectDefinition/data
      value: '{{hub copyConfigMapData "ztp-group" "oadp-cm" hub}}'
```

### Enforcing the Policies

To enforce the stage policies for performing IBU, create a ClusterGroupUpgrade (CGU) CR for each stage policy. The `group-ibu-oadp-cm-policy` policy, which distributes the OADP configmap to applicable managed clusters, should be included in the Prep CGU along with `group-ibu-prep-policy`. Since the OADP configmap should be propagated prior to transitioning the IBU stage to `Prep`, it must be the first policy in the Prep CGU.

For more detailed information on using the Life Cycle Agent (LCA) operator, refer to the [docs](https://github.com/openshift-kni/lifecycle-agent/tree/main/docs).
