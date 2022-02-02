Installation
-
1. Use SiteConfig to generate MachineConfig for disk partitioning. Make sure to modify values in the mc appropriately as it is dependent on the underlying disk.
```yaml
nodes:
  - diskPartition:
    - device: /dev/sda
      partitions:
       - mount_point: /var/imageregistry
         size: 102500
         start: 344844
```
   
3. Use PGT, to apply the following to create the pv and pvc and patch imageregistry config as part of normal day-2 operation
   ```yaml
   sourceFiles:
     - fileName: StoragePVC.yaml
       policyName: "pvc-for-image-registry"
       metadata:
         name: image-registry-pvc
         namespace: openshift-image-registry
       spec:
         accessModes:
           - ReadWriteMany
         resources:
           requests:
             storage: 100Gi
         storageClassName: image-registry-sc
         volumeMode: Filesystem
     - fileName: ImageRegistryPV.yaml
       policyName: "pv-for-image-registry"
     - fileName: ImageRegistryConfig.yaml
       policyName: "config-for-image-registry"
       spec:
         storage:
           pvc:
             claim: "image-registry-pvc"
   ```

Verify/Debug
-
- Check the CRD `Config` of group `imageregistry.operator.openshift.io`'s instance `cluster` is not reporting any error
- Within a few minutes after the installation process is complete you should see the pvc filling up.
- From inside the node:
  - Successful login to the registry with podman:
     ```
     oc login -u kubeadmin -p <password_from_install_log> https://api-int.<cluster_name>.<base_domain>:6443
     podman login -u kubeadmin -p $(oc whoami -t) image-registry.openshift-image-registry.svc:5000
     ```
  - Check for disk partitioning:
    ```
    [core@mysno ~]$ lsblk
    NAME   MAJ:MIN RM   SIZE RO TYPE MOUNTPOINT
    sda      8:0    0 446.6G  0 disk
      |-sda1   8:1    0     1M  0 part
      |-sda2   8:2    0   127M  0 part
      |-sda3   8:3    0   384M  0 part /boot
      |-sda4   8:4    0 336.3G  0 part /sysroot
      `-sda5   8:5    0 100.1G  0 part /var/imageregistry
    sdb      8:16   0 446.6G  0 disk
    sr0     11:0    1   104M  0 rom
    ```


Additional Resources
-

- For more info on using image registry operator check the [official docs](https://docs.openshift.com/container-platform/4.10/registry/index.html).
  - You can also expose the registry to outside world, make it secure and so on
