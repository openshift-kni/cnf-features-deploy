## Steps to Upgrade from v4.9 [argocd/example](https://github.com/openshift-kni/cnf-features-deploy/tree/release-4.9/ztp/gitops-subscriptions/argocd/resource-hook-example) to v4.10 [argocd/example](https://github.com/openshift-kni/cnf-features-deploy/tree/master/ztp/gitops-subscriptions/argocd/example)

Follow the steps 1, 2 and 3 to prepare the hub cluster as [Readme](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/gitops-subscriptions/argocd/README.md) (Preparation of Hub cluster) section mentioned.
 
**Upgrading the SiteConfig repository:**
In the siteConfig repository follow the steps below:
 1. Delete the post-sync.yaml and pre-sync.yaml files.
 1. If the siteconfig*.yaml in the repository contain definition for the Namespace remove it.
 1. Update the kustomization.yaml file to include all SiteConfig yaml files in the `generators` section as below
```
generators:
# list of all the siteConfig*.yaml exist in the repository
- site1.yaml
- site2.yaml
- site3.yaml
``` 
 Save and push changes to the repository. The ArgoCD application will detect the changes that have been done to the SiteConfig git repository   
 and re-sync the cluster installation CRS to the hub cluster. Note: the installed clusters will not be re provisioned.
 
**Upgrading the policyGenTemplate repository:**
In the policyGenTemplate repository follow the steps below:
 1. Delete the post-sync.yaml and pre-sync.yaml files.
 1. If any PolicyGenTemplate*.yaml in the repository contains a namespace definition, remove it and define the Namespace in separate file similar to the new [example/ns.yaml](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/gitops-subscriptions/argocd/example/policygentemplates/ns.yaml).
 1. Adjust all Namespace names to start with prefix  `ztp`.
 1. Adjust the PolicyGenTemplate*.yaml Namespace accordingly. 
 1. Update the kustomization.yaml file to have all PGTs in the `generators` section, and all Namespaces in the `resources` section. For example:
```
generators:
# List of all the PolicyGenTemplate*.yaml exist in the repository
- common-ranGen.yaml
- group-du-sno-ranGen.yaml
- example-sno-site.yaml

resources:
# List of Namespace*.yaml required by the PolicyGenTemplate*.yaml
- ns.yaml
``` 
 Save and push changes to the repository. The ArgoCD application will detect the changes that have been done to the PolicyGenTemplates git repository   
 and re-sync the ACM policies to the hub cluster. Note: The ACM policies will be recreated under the new Namesapces.


For Troubleshooting follow the [Readme](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/gitops-subscriptions/argocd/README.md) Troubleshooting section.
 