# GitOps ZTP pipeline
![GitOps ZTP flow overview](ztp_gitops_flow.png)

## Installing the GitOps Zero Touch Provisioning pipeline

### Preparation of ZTP GIT repository
Create a GIT repository for hosting site configuration data. The ZTP pipeline will require read access to this repository.
1. Create a directory structure with separate paths for SiteConfig and PolicyGenTemplate CRs
2. Export the argocd directroy from the ztp-site-generator container image by executing the following commands:
```
    $ podman pull quay.io/redhat_emp1/ztp-site-generator:latest
    $ mkdir -p ./out
    $ podman run --rm ztp-site-generator:latest extract /home/ztp --tar | tar x -C ./out
```
3. Check the out directory that created above. It contains the following sub directories
  - out/extra-manifest: contains the source CRs files that SiteConfig uses to generate extra manifest configMap.
  - out/source-crs: contains the source CRs files that PolicyGenTemplate uses to generate the ACM policies.
  - out/argocd/deployment: contains patches and yaml file to apply on the hub cluster for use in the next step of this procedure.
  - out/argocd/example: contains example SiteConfig and PolicyGenTemplate that represent our recommended configuration.

### Preparation of Hub cluster for ZTP
These steps configure your hub cluster with a set of ArgoCD Applications which generate the required installation and policy CRs for each site based on a ZTP gitops flow.

**Requirements:**
- Openshift Cluster v4.8/v4.9 as Hub cluster
- Advanced Cluster Management (ACM) operator v2.3/v2.4 installed in the hub cluster
- Red Hat OpenShift GitOps operator v1.3 on the hub cluster

**Steps:**
1. Install the [Topology Aware Lifecycle Operator](https://github.com/openshift-kni/cluster-group-upgrades-operator#readme), which will coordinate with any new sites added by ZTP and manage application of the PGT-generated policies.
2. Patch the ArgoCD instance in the hub cluster using the patch files previously extracted into the out/argocd/deployment/ directory:
```
    $ oc patch argocd openshift-gitops -n openshift-gitops  --patch-file out/argocd/deployment/argocd-openshift-gitops-patch.json --type=merge
    $ oc patch deployment openshift-gitops-repo-server -n openshift-gitops --patch-file out/argocd/deployment/deployment-openshift-repo-server-patch.json
```
3. Prepare the ArgoCD pipeline configuration
- Create a git repository with directory structure similar to the example directory.
- Configure access to the repository using the ArgoCD UI. Under Settings configure:
  - Repositories --> Add connection information (URL ending in .git, eg https://repo.example.com/repo.git, and credentials)
  - Certificates --> Add the public certificate for the repository if needed
- Modify the two ArgoCD Applications (out/argocd/deployment/clusters-app.yaml and out/argocd/deployment/policies-app.yaml) based on your GIT repository:
  - Update URL to point to git repository. The URL must end with .git, eg: https://repo.example.com/repo.git
  - The targetRevision should indicate which branch to monitor
  - The path should specify the path to the SiteConfig or PolicyGenTemplate CRs respectively
4. Apply pipeline configuration to your *hub* cluster using the following command.
```
    oc apply -k out/argocd/deployment
```

### Deploying a site
The following steps prepare the hub cluster for site deployment and initiate ZTP by pushing CRs to your GIT repository.
1. Create the required secrets for site. These resources must be in a namespace with a name matching the cluster name. In out/argocd/example/siteconfig/example-sno.yaml the cluster name & namespace is `example-sno`
   1. Create a pull secret for the cluster. The pull secret must contain all credentials necessary for installing OpenShift and all required operators. In all of the example SiteConfigs this is named `assisted-deployment-pull-secret`
   2. Create a BMC authentication secret for each host you will be deploying.
   3. Create the appropriate namespace and manually apply these secrets to on your hub cluster before proceeding.
2. Create a SiteConfig CR for your cluster in your local clone of the git repository:
   1. Begin by choosing an appropriate example from out/argocd/example/siteconfig/.  There are examples there for SNO, 3-node, and standard clusters.
   2. Change the cluster and host details in the example to match your desired cluster.  Some important notes:
      - The clusterImageSetNameRef must match an imageset available on the hub cluster (run `oc get clusterimagesets` for the list of supported versions on your hub)
      - Ensure the cluster networking sections are defined correctly:
        - For SNO deployments, you must define a `MachineNetwork` section and not the `apiVIP` and `ingressVIP` values.
        - For 3-node and standard deployments, you must define the `apiVIP` and `ingressVIP` values and not the `MachineNetwork` section.
      - The set of cluster labels that you define in the `clusterLabels` section must correspond to the PolicyGenTemplate labels you will be defining in a later step.
      - Ensure you have updated the hostnames, BMC address, BMC secret name, network configuration sections
      - Ensure you have the required number of host entries defined:
        - For SNO deployments, you must have exactly one host defined.
        - For 3-node deployments, you must have exactly three hosts defined.
        - For standard deployments, you must have exactly three hosts defined with `role: master` and one or more hosts defined with `role: worker`
      - The default set of extra-manifest MachineConfigs can be inspected in out/argocd/extra-manifest, and will be automatically applied to the cluster as it is installed.
        - Optional: For provisiong additional install-time manifests on the provisioned cluster, create a directory in your GIT repository (for example, `sno-extra-manifest/`) and add your custom manifest CRs to this directory.  If your SiteConfig.yaml refers to this directory via the `extraManifestPath` field, any CRs in this referenced directory will be appended to the default set of extra manifests.
   3. Add the SiteConfig CR to the kustomization.yaml in the 'generators' section, much like in the example out/argocd/example/siteconfig/kustomization.yaml
   4. Commit your SiteConfig and associated kustomization.yaml in git.
3. Create the PolicyGenTemplate CR for your site in your local clone of the git repository:
   1. Begin by choosing an appropriate example from out/argocd/example/policygentemplates. This directory demonstrates a 3-level policy framework which represents a well-supported low-latency profile tuned for the needs of 5G Telco DU deployments:
      - A single `common-ranGen.yaml` that should apply to all types of sites
      - A set of shared `group-du-*-ranGen.yaml`, each of which should be common across a set of similar clusters
      - An example `example-*-site.yaml` which will normally be copied and updated for each individual site
   2. Ensure the labels defined in your PGTs `bindingRules` section correspond to the proper labels defined on the SiteConfig file(s) of the clusters you are managing.
   3. Ensure the content of the overlaid spec files matches your desired end state.  As a reference, the out/source-crs directory contains the full list of source-crs available to be included and overlayed by your PGT templates.
      - Note: Depending on the specific requirements of your clusters, you may need more than just a single group policy per cluster type, especially considering the example group policies each has a single PerformancePolicy which can only be shared across a set of clusters if those clusters consist of identical hardware configurations.
   4. Define all the policy namespaces in a yaml file much like in the example out/argocd/example/policygentemplates/ns.yaml
   5. Add all the PGTs and ns.yaml to the kustomization.yaml file, much like in the example out/argocd/example/policygentemplates/kustomization.yaml
   6. Commit the PolicyGenTemplate CRs, ns.yaml, and associated kustomization.yaml in git.
4. Push your changes to the git repository, and the ArgoCD pipeline will detect the changes and begin the site deployment. The SiteConfig and PolicyGenTemplate CRs may be pushed simultaneously.

### Monitoring progress
The ArgoCD pipeline uses the SiteConfig and PolicyGenTemplate CRs in GIT to generate the cluster configuration CRs & ACM policies then sync them to the hub.

The progress of this synchronization can be monitored in the ArgoCD dashboard.

Once the synchonization is complete, the installation generally proceeds in two phases:

1. The Assisted Service Operator installs OpenShift on the cluster

The progress of cluster installation can be monitored from the ACM dash board, or the command line:
```
     $ export CLUSTER=<clusterName>
     $ oc get agentclusterinstall -n $CLUSTER $CLUSTER -o jsonpath='{.status.conditions[?(@.type=="Completed")]}' | jq
     $ curl -sk $(oc get agentclusterinstall -n $CLUSTER $CLUSTER -o jsonpath='{.status.debugInfo.eventsURL}')  | jq '.[-2,-1]'
```

2. The Topology Aware Lifecycle Operator(TALO) then applies the configuration policies which are bound to the cluster

After the cluster installation is completed and cluster becomes `Ready`, a ClusterGroupUpgrade CR corresponding to this cluster, with a list of ordered policies defined by the ran.openshift.io/ztp-deploy-wave annotations, will be automatically created by TALO. The cluster's policies will be applied in the order listed in ClusterGroupUpgrade CR.

The high-level progress of configuration policy reconciliation can be monitored via the command line:
```
     $ export CLUSTER=<clusterName>
     $ oc get clustergroupupgrades -n ztp-install $CLUSTER -o jsonpath='{.status.conditions[?(@.type=="Ready")]}'
```

The detailed policy compliant status can be monitored in the ACM dash board, or the command line:
```
     $ oc get policies -n $CLUSTER
```

The final policy that will become compliant is the one defined in the `*-du-validator-policy` policies. This policy, when compliant on a cluster, ensures that all cluster configuration, operator installation, and operator configuration has completed.

After all policies become complaint, `ztp-done` label will be added to the cluster that indicates the whole ZTP pipeline has completed for the cluster.
```
     $ oc get managedcluster $CLUSTER -o jsonpath='{.metadata.labels}' | grep ztp-done
```

### Site Cleanup
To remove a site and the associated installation and configuration policy CRs by removing the SiteConfig & PolicyGenTemplate file name from the kustomization.yaml file. The generated CRs will be removed as well.
**NOTE: After removing the SiteConfig file, if its corresponding clusters stuck in the detach process check [ACM page](https://access.redhat.com/documentation/en-us/red_hat_advanced_cluster_management_for_kubernetes/2.4/html/clusters/managing-your-clusters#remove-managed-cluster) how to clean detach managed cluster **

### Pipeline Teardown
If you need to remove the ArgoCD pipeline and all generated artifacts follow this procedure
1. Detach all clusters from ACM
1. Delete the kustomization.yaml under deployment directory
```
    $ oc delete -k out/argocd/deployment
```

## Troubleshooting GitOps ZTP
As noted above the ArgoCD pipeline uses the SiteConfig and PolicyGenTemplate CRs from GIT to generate the cluster configuration CRs & ACM policies. The following steps can be used to troubleshoot issues that may occur in this process.

### Validate generation of installation CRs
The installation CRs are applied to the hub cluster in a namespace with name matching the site name.  
```
    $ oc get AgentClusterInstall -n <clusterName>
```
If no object is returned, troubleshoot the ArgoCD pipeline flow from SiteConfig to installation CRs.

1. Did the SiteConfig->ManagedCluster get generated to the hub cluster ?
```
     $ oc get managedcluster
```
If the SiteConfig->ManagedCluster is missing, check the `clusters` application failed to synchronize the files from GIT to the hub.
```
    $ oc describe -n openshift-gitops application clusters 
```

Check for `Status: Conditions:` it will show error logs ex; setting invalid `extraManifestPath: ` in the siteConfig will raise error as below.
```
Status:
  Conditions:
    Last Transition Time:  2021-11-26T17:21:39Z
    Message:               rpc error: code = Unknown desc = `kustomize build /tmp/https___git.com/ran-sites/siteconfigs/ --enable-alpha-plugins` failed exit status 1: 2021/11/26 17:21:40 Error could not create extra-manifest ranSite1.extra-manifest3 stat extra-manifest3: no such file or directory
2021/11/26 17:21:40 Error: could not build the entire SiteConfig defined by /tmp/kust-plugin-config-913473579: stat extra-manifest3: no such file or directory
Error: failure in plugin configured via /tmp/kust-plugin-config-913473579; exit status 1: exit status 1
    Type:  ComparisonError
```

Check for `Status: Sync:` if there are log errors the `Sync: Status:` could be as below `Unknown` / `Error`.
```
Status:
  Sync:
    Compared To:
      Destination:
        Namespace:  clusters-sub
        Server:     https://kubernetes.default.svc
      Source:
        Path:             sites-config
        Repo URL:         https://git.com/ran-sites/siteconfigs/.git
        Target Revision:  master
    Status:               Unknown
```
 

### Validate generation of configuration policy CRs

Policy CRs are generated in the same namespace as the PolicyGenTemplate from which they were created. The same troubleshooting flow applies to all policy CRs generated from PolicyGenTemplates regardless of whether they are ztp-common, ztp-group or ztp-site based.  
```
    $ export NS=<namespace>
    $ oc get policy -n $NS
```
The expected set of policy wrapped CRs should be displayed.

If the Policies failed synchronized follow these troubleshooting steps:

```
    $ oc describe -n openshift-gitops application policies 
```

1. Check for `Status: Conditions:` which will show error logs. For example, setting an invalid `sourceFile->fileName:` will generate an error as below. 
```
Status:
  Conditions:
    Last Transition Time:  2021-11-26T17:21:39Z
    Message:               rpc error: code = Unknown desc = `kustomize build /tmp/https___git.com/ran-sites/policies/ --enable-alpha-plugins` failed exit status 1: 2021/11/26 17:21:40 Error could not find test.yaml under source-crs/: no such file or directory
Error: failure in plugin configured via /tmp/kust-plugin-config-52463179; exit status 1: exit status 1
    Type:  ComparisonError
```

1. Check for `Status: Sync:`. If there are log errors at `Status: Conditions:`, the `Sync: Status:` will be as `Unknown` or `Error`.
```
Status:
  Sync:
    Compared To:
      Destination:
        Namespace:  policies-sub
        Server:     https://kubernetes.default.svc
      Source:
        Path:             policies
        Repo URL:         https://git.com/ran-sites/policies/.git
        Target Revision:  master
    Status:               Error
```

1. Did the policies get copied to the cluster namespace?
When ACM recognizes that policies apply to a ManagedCluster, the policy CR objects are applied to the cluster namespace.
```
    $ oc get policy -n <clusterName>
```
All applicable policies should be copied here by ACM (ie should show common, group and site policies). The policy names are `<policyGenTemplate.Name>.<policyName>`

1. For any policies not copied to the cluster namespace check the placement rule.
The matchSelector in the PlacementRule for those policies should match labels on the ManagedCluster.
```
    $ oc get placementrule -n $NS`
```
Make note of the PlacementRule name appropriate for the missing policy (eg common, group or site)
```
    $ oc get placementrule -n $NS <placmentRuleName> -o yaml
```
- The status/decisions should include your cluster name
- The key/value of the matchSelector in the spec should match the labels on your managed cluster.
Check labels on MangedCluster:  
```
    $ oc get ManagedCluster $CLUSTER -o jsonpath='{.metadata.labels}' | jq
```
1. Are some policies compliant and others not
```
    $ oc get policy -n $CLUSTER
```
If the Namespace, OperatorGroup, and Subscription policies are compliant but the operator configuration policies are not it is likely that the operators did not install on the spoke cluster. This causes the operator config policies to fail to apply because the CRD is not yet applied to spoke.
