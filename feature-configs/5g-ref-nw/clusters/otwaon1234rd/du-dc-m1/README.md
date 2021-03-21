# Data center DU pool
This is an example of a data center DU pool configurations
## Structure
The manifest overlays structure can be seen from the `kustomize.yaml`:

```yml
resources:
- role-hardware
- nw-config
```

The file is logically divided into two parts: operators and customizations. The operator resources directly reference the [operators](../../../operators) folder components relevant to the role. This is done to avoid mutations by nameSuffix transformers used inside the Customizations section resources.

The Customizations section includes the adjustments required to the specific pool instance, separated to `role-hardware` and `nw-config`. This separation is required for two reasons:
1. Enables a workaround to the [MCO race condition](https://bugzilla.redhat.com/show_bug.cgi?id=1916169)
2. Provides an option for additional network configurations for the same role-hardware pool. Please note that network node selectors are separate from the role-hardware ones.
 
The role-hardware folder contains the following components:
- performance
  - Overlays [profile-base/su-large/performance](../../../profile-base/du-dual/performance)
  - Adds a suffix to the performance profile name (to allow multiple pools with different performance profiles in the cluster)
  - Mutates the node selector
- ptp
  - Overlays [profile-base/du-general/ptp](../../../profile-base/du-general/ptp)
  - Adds a suffix to the performance profile name
  - Mutates the PTP interface name
- sctp
  - Overlays [profile-base/ran-general](../../../profile-base/ran-general)
  - Modifies SCTP node selector

The nw-config folder:
  - Overlays [profile-base/du-dual/sriov](../../../profile-base/du-dual/sriov)
  - Defines network namespace and adds it to all instances of `SriovNetwork` resource
  - Mutates the names of all the resources in this pool by adding a name suffix
  - Mutates node selectors, VLAN IDs and resource names of all the instances of `SriovNetwork` and `SriovNetworkNodePolicy`

## Labeling
Working nodes in this pool must be labeled as follows:
- `node-role.kubernetes.io/worker-du-dual-otwaon1234rd-m1=""` - used for machine config pool
- `ptp/slave=""` - used as node selector for PTP
- `ran.example.com/worker-du-dual-otwaon1234rd-m1=""` - used as node selector for SR-IOV networks
Please note the separate node selectors for machine config pool and SR-IOV networks. This is done to allow separate network configurations for different nodes in the same hardware-role pool of machines