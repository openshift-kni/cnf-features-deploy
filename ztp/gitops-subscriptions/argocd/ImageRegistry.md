Installation
-
1. Use SiteConfig to generate MachineConfig for disk partitioning. Make sure to modify values in the mc appropriately as it is dependent on the underlying disk.   

   It is important use persistent naming for device, especially if you have more than one disk as names like `/dev/sda` and `/dev/sdb` may switch at every reboot. You can use `rootDeviceHints` to choose the bootable device and then use same the device for further partitioning, in this case, for Image registry. More info on persistent naming [here](https://wiki.archlinux.org/title/persistent_block_device_naming#Persistent_naming_methods).

   wwn is used below. 


```yaml
nodes:
  - rootDeviceHints:
      wwn: "0x62cea7f05c98c2002708a0a22ff480ea"
    diskPartition:
    - device: /dev/disk/by-id/wwn-0x62cea7f05c98c2002708a0a22ff480ea # depends on the hardware. can also serial num or device name. recommend using wwn. match with the rootDeviceHint above
      partitions:
       - mount_point: /var/imageregistry
         size: 102500 # min value 100gig for image registry
         start: 344844 # 25000 min value, otherwise root partition is too small. Recommended startMiB (Disk - sizeMiB - some buffer), if startMiB + sizeMiB  exceeds disk size, installation will fail
```
   
3. Use PGT, to apply the following to create the pv and pvc and patch imageregistry config as part of normal day-2 operation. Select the appropriate PGT for each source-cr and refer to `wave` doc for more help. Below is as example if you would like to test it at the site level.
   ```yaml
   sourceFiles:
      # storage class
      - fileName: StorageClass.yaml
        policyName: "sc-for-image-registry"
        metadata:
          name: image-registry-sc
          annotations:
            ran.openshift.io/ztp-deploy-wave: "100" # remove this when moved to the right PGT (site/group/common)
     # persistent volume claim
     - fileName: StoragePVC.yaml
       policyName: "pvc-for-image-registry"
       metadata:
         name: image-registry-pvc
         namespace: openshift-image-registry
         annotations:
            ran.openshift.io/ztp-deploy-wave: "100"  # remove this when moved to the right PGT (site/group/common)
       spec:
         accessModes:
           - ReadWriteMany
         resources:
           requests:
             storage: 100Gi
         storageClassName: image-registry-sc
         volumeMode: Filesystem
      # persistent volume
     - fileName: ImageRegistryPV.yaml # this is assuming that mount_point is set to `/var/imageregistry` in SiteConfig
                                      # using StorageClass `image-registry-sc` (see the first sc-file)
       policyName: "pv-for-image-registry" 
       metadata:
         annotations:
           ran.openshift.io/ztp-deploy-wave: "100"  # remove this when moved to the right PGT (site/group/common)
      # configure registry to point to the pvc created above
     - fileName: ImageRegistryConfig.yaml
       policyName: "config-for-image-registry"
       complianceType: musthave # do not use `mustlyonlyhave` as it will cause deployment failure of registry pod.
       metadata:
         annotations:
           ran.openshift.io/ztp-deploy-wave: "100"  # remove this when moved to the right PGT (site/group/common)
       spec:
         storage:
           pvc:
             claim: "image-registry-pvc"
   ```

Verify/Debug
-
- Check the CRD `Config` of group `imageregistry.operator.openshift.io`'s instance `cluster` is not reporting any error
- Within a few minutes after the installation process is complete you should see the pvc filling up.
- Check "registry*" pod is up correctly under `openshift-image-registry` namespace
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
