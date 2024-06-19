# Accelerated Zero Touch Provisioning

From OpenShift Container Platform 4.16, you can configure accelerated provisioning of GitOps ZTP for single-node OpenShift (SNO) to reduce the time taken for installation. Accelerated ZTP speeds up installation by applying Day-2 manifests derived from policies at an earlier stage. 

## Activating accelerated ZTP
You can activate accelerated ZTP using the `spec.clusters.clusterLabels.accelerated-ztp` label in `SiteConfig` CR, as in the following example:

```yaml
clusterLabels:
   ...
   accelerated-ztp: full
   ...
```
Use `accelerated-ztp: full` to fully automate the accelerated process. GitOps ZTP updates the ‘AgentClusterInstall’ resource with a reference to the accelerated GitOps ZTP ConfigMap, and includes policies extracted from TALM, and accelerated GitOps ZTP job manifests. 
If you use `accelerated-ztp: partial`, GitOps ZTP does not include the accelerated GitOps ZTP job manifests, but includes policy-derived objects of the following kinds, created during the cluster installation:

```bash
"PerformanceProfile.performance.openshift.io"
"Tuned.tuned.openshift.io"
"Namespace"
"CatalogSource.operators.coreos.com"
"ContainerRuntimeConfig.machineconfiguration.openshift.io" 
```
This partial acceleration reduces the number of reboots done by the node when applying resources of the kind `Performance Profile`, `Tuned` and `ContainerRuntimeConfig`. 
TALM installs the operator subscriptions derived from policies after ACM completes the import of  the cluster, following the same flow as standard, non-accelerated ZTP.

## The accelerated ZTP process

Accelerated ZTP uses an additional `ConfigMap` to create the resources derived from policies on the spoke cluster (the first `ConfigMap` includes manifests that the GitOps ZTP workflow uses to customize cluster installation). TALM creates a second `ConfigMap` once it detects that the `accelerated-ztp` label is set. As part of accelerated ZTP, the `SiteConfig` generator adds a reference to that second `ConfigMap` using the naming convention “<spoke-cluster-name>-aztp”. 
After TALM creates the `<spoke-cluster-name>-aztp` `ConfigMap`, it finds all policies bound to the managed cluster and extracts the ZTP profile information. TALM adds the ZTP profile information to the `<spoke-cluster-name>-aztp` `ConfigMap` CR and applies the CR to the hub cluster API. 

