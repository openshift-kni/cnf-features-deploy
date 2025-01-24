# Disk encryption with PCR protection:

## Configuring initial Disk encryption with PCR:
Enable disk encryption with PCR 1 and 7: configuration done via siteconfig diskEncryption->tpm2 object (PCR 1 and 7 supported), for instance:
```
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "encryption-tpm2"
  namespace: "encryption-tpm2"
spec:
  clusters:
  - clusterName: "encryption-tpm2"
    clusterImageSetNameRef: "openshift-v4.13.0"
    diskEncryption:
      type: "tpm2"
      tpm2:
        pcrList: "1,7"
    nodes:
      - hostName: "node1"
        role: master
```

## Upgrade support
PCR disk protection is disabled while the upgrade is in progress:
* Upgrades in progress are detected before the host is restarted.
* When an upgrade is detected an additional “reserved” key is added that does not use PCR, essentially disabling PCR protection.
* Upon reboot, if the “reserved” pin is present, it is removed in order to re-enable PCR protection. 
* This cycle of disabling PCR protection right before reboot and re-enabling it right after the host is finished booting continues until the upgrade completes

## Upgrade detection plugins supported
* file: create a file at a given location(e.g. /etc/host-hw-Updating.flag). Upgrade is detected if the file is present.
* linux fwupd tool: detecting firmware upgrades (Bios, device firmwares, …) with the efibootmgr command. An upgrade is detected if the next boot will boot the firmware updater, as indicated by “BootNext” set to fwupd.
* ostree: an upgrade is detected if the output of ostree admin status contains “staged” ot “pending”
* talm: if the host is deployed by ZTP, the host can check the status of the TALM lifecycle managed to understand is an upgrade is in progress. The host retrieves its managedCluster CR object from the Hub cluster. an upgrade is detected if the managed cluster is labeled with the ztp-running label.