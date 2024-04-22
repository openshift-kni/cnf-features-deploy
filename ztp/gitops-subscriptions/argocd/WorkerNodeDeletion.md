# Delete and re-provision a worker node using ZTP
Starting from ACM 2.8, it supports GitOps workflow to cleanly delete a node from an existing cluster by deleting the BMH CR on the hub cluster that is annotated for cleanup. This document provides guidance on how to delete and re-provision a worker node using ZTP workflow.

## Prerequisites
1. A spoke cluster (ie. Standard, Compact+workers, sno+workers) installed and configured using the GitOps ZTP flow, as described in [GitOps ZTP flow](README.md)
1. ACM 2.8+ with MultiClusterHub created and configured, running on OCP 4.13+ bare metal cluster

## Delete a worker node from spoke cluster
1. Annotate the BMH CR of the worker node with the "bmac.agent-install.openshift.io/remove-agent-and-node-on-delete=true"    annotation. Add the annotation via SiteConfig as the following, then push the changes to git repo and wait for the BMH CR on the hub cluster has the annotation applied.
    ```yaml
    nodes:
    - hostname: node6
      role: "worker"
      crAnnotations:
        add:
          BareMetalHost:
            bmac.agent-install.openshift.io/remove-agent-and-node-on-delete: true
    ```
2. Delete the BMH CR of the worker node that has been annotated. Suppress the generation of the BMH CR via SiteConfig as the following, then push the changes to git repo and wait for deprovision to start.
   ```yaml
   nodes:
   - hostname: node6
     role: "worker"
     crSuppression:
       - BareMetalHost
   ```
3. The status of the BMH CR should be changed to "deprovisioning". Wait for the BMH to finish deprovisioning, and to be fully deleted.

## Verify the node is deleted
1. Verify the BMH and Agent CRs for the worker node have been deleted from the hub cluster.
```shell
oc get bmh -n <cluster-ns>
oc get agent -n <cluster-ns>
```
2. Verify the node record has been deleted from the spoke cluster.
```shell
oc get nodes
``` 

## Reprovision the worker node
Delete the following changes from the SiteConfig you added previously for the node deletion, then push the changes to the git repo and wait for sync to complete. This will re-generate the BMH CR of the worker node and trigger the re-install of the node.
```yaml
   nodes:
   - hostname: node6
     role: "worker"
     crAnnotations:
       add:
         BareMetalHost:
           bmac.agent-install.openshift.io/remove-agent-and-node-on-delete: true
     crSuppression:
       - BareMetalHost
```