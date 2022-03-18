# Enable and Verify for Secure Boot for SNO

## Enabling Secure Boot 

 - Configure `SiteConfig` with `UEFIScureBoot` for `bootMode`. E.g part of yaml below 
   ```yaml
    nodes:
    - hostName: "myhost"
      bootMode: "UEFISecureBoot"
   ```

- This value should now be available in the Hub (cluster w/ ACM) cluster's appropriate BareMetalHost (bmh) CR. e.g part of the yaml below 
  ```yaml
  spec:
    bootMode: UEFISecureBoot
  ```

## Verifying Secure Boot

1. log into the node

    - With `oc`
      ```shell
      oc debug node/myhost.rh.com
      
      # don't forget to link the logs 
      sh-4.4#  chroot /host
      ```
  
    - **With `ssh` command** 
       ```shell
       ssh -i key core@myhost.rh.com # key: private key associated with the node
       ```
  
2. Once logged into the node there are a couple of ways to verify. 

    - **With `journalctl`**
      ```shell
      sh-4.4# journalctl -g secureboot
      -- Logs begin at Wed 2022-03-23 17:14:14 UTC, end at Fri 2022-03-25 16:46:26 UTC. --
      Mar 23 17:14:14 localhost kernel: secureboot: Secure boot enabled
      -- Reboot --
      Mar 23 17:20:00 localhost kernel: secureboot: Secure boot enabled
      -- Reboot --
      Mar 23 17:54:41 localhost kernel: secureboot: Secure boot enabled
      -- Reboot --
      Mar 23 18:04:16 localhost kernel: secureboot: Secure boot enabled
      ```

    - **With `mokutil`**
      ```shell
      mokutil --sb-state
      SecureBoot enabled
      ```

**Debugging mokutil**

If you're not using the latest set of source-crs and are running the real-time kernel, `mokutil` may return the following error message:

```shell
mokutil --sb-state
EFI variables are not supported on this system
```

There are multiple ways to enable kernel access to the EFI variables that are required by `mokutil`: 

- Update to the latest set of source-cr

- Append `PerformanceProfile.yaml`'s  `additionalKernelArgs` from PGT

  ```yaml
  spec:
    additionalKernelArgs:
      - ...
      - "efi=runtime"
  ```
- Use SiteConfig's `sno-extra-manifest` feature.

  1. Create MachineConfig CR `99-efi-runtime-path-kargs.yaml`

   ```yaml
   apiVersion: machineconfiguration.openshift.io/v1
   kind: MachineConfig
   metadata:
     labels:
       machineconfiguration.openshift.io/role: master
     name: 99-efi-runtime-path-kargs
   spec:
     kernelArguments:
     - "efi=runtime"
  ```
  2. Include the `MC` with your `SiteConfig` as part of extra manifest. E.g part of the CR below
  
     ```yaml
     clusters:
     - clusterName: "myhost"
       extraManifestPath: myhost/sno-extra-manifest/
     ```

     File dir may look like below 

     ```shell
     ➜ tree .
       └── install
       ├── myhost
       │   ├── myhost.yaml
       │   ├── secret.yaml
       │   └── sno-extra-manifest
       │       └── 99-efi-runtime-path-kargs.yaml
       └── kustomization.yaml

     ```

##More info on Secure Boot
- [Blog](https://cloud.redhat.com/blog/validating-secure-boot-functionality-in-a-sno-for-openshift-4.9)