## Image-Based Upgrades (IBU)
This directory contains examples of generating resources required for Image Based Upgrades (IBU) utilizing the [Life Cycle Agent operator](https://github.com/openshift-kni/lifecycle-agent). These examples define policies to automate image-based upgrades, ensuring seamless deployment across managed clusters through Gitops.


### Prerequisites

Before using the IBU examples, ensure that the following namespaces have been created:

- `ztp-common`: The root policy will be created in this namespace. If you use another name for the `common` namespace, please remember to update the `example-oadp-policy.yaml` policy.
- `openshift-adp`: The ConfigMap containing the related OpenShift API for Data Protection (OADP) Custom Resources (CRs) will be copied to this namespace on the applicable spoke cluster(s).

### Setup ArgoCD Application

To deploy the IBU examples, you can use the existing [policies-app](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/gitops-subscriptions/argocd/deployment/policies-app.yaml), which is also used for deploying DU profile policy examples. Refer to the [ReadMe](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/gitops-subscriptions/argocd/README.md) section "Preparation of Hub cluster for ZTP" for detailed instructions on setting up the ArgoCD policies application.

Ensure that your Git repository, which will be used with the ArgoCD policies application, contains a directory structured as follows:

```plaintext
├── source-crs/
│   ├── ibu/
│   │    ├── PlatformBackupRestore.yaml
├── ...
├── custom-oadp-workload-crs.yaml
├── example-oadp-policy.yaml
├── kustomization.yaml
```

Note that [`source-crs/ibu/PlatformBackupRestore.yaml`](https://github.com/openshift-kni/cnf-features-deploy/tree/master/ztp/source-crs/ibu/PlatformBackupRestore.yaml) is provided in the ZTP image, however, it is important to ensure that the `kustomization.yaml` file is located in the same directory structure shown above in order to reference the `PlatformBackupRestore.yaml` manifest.

### Generating the ConfigMap and Policy

Use the [`configMapGenerator`](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/#configmapgenerator) provided by the Kustomize tool to generate a ConfigMap encapsulating the OADP CRs for IBU. The OpenShift platform-related CRs are available [here](https://github.com/openshift-kni/cnf-features-deploy/tree/master/ztp/source-crs/ibu/PlatformBackupRestore.yaml).

Additional workload OADP CRs may be included in the ConfigMap generation process as shown in the example below:

```yaml
configMapGenerator:
- files:
  - source-crs/ibu/PlatformBackupRestore.yaml
  - custom-oadp-workload-crs.yaml
  name: oadp-cm
  namespace: ztp-common

generatorOptions:
  disableNameSuffixHash: true
```

Ensure that the `custom-oadp-workload-crs.yaml` file includes a one-to-one mapping of OADP backup and restore CRs. It's important to note that these CRs can be stored either in separate YAML manifests or consolidated within a single YAML file (as shown below), with each CR section separated by the `---` directive.

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

> [!IMPORTANT]
> Don't forget to create and include the policy [example-oadp-policy.yaml](./example-oadp-policy.yaml) under the `resources` object in the `kustomization.yaml` file.
> The policy uses hub-side templating and requires Advanced Cluster Management (ACM) 2.8 and later versions for the `copyConfigMapData` function. However, it is important to note that 2.9.2 will be the minimum supported ACM version for IBU.


### Enforcing the Policy

To enforce the policy and distribute the ConfigMap to the applicable managed clusters, create a ClusterGroupUpgrade (CGU) CR referencing the aforementioned policy. This should be done prior to setting the IBU stage to `Prep`.

For more detailed information on using the Life Cycle Agent (LCA) operator, refer to the [docs](https://github.com/openshift-kni/lifecycle-agent/tree/main/docs).
