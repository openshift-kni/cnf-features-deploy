# Remote worker node DU
This is an example of a DU RWN configuration
## Structure
The structure is identical to [du-dual-m1](../otwaon1234rd/du-dual-m1).
## Labeling
Working nodes in this pool must be labeled as follows:
- `node-role.kubernetes.io/worker-du-single-otwaon23456rw=""` - used for machine config pool
- `ptp/slave=""` - used as node selector for PTP
- `ran.example.com/worker-du-single-otwaon23456rw=""` - used as node selector for SR-IOV networks
Please note the separate node selectors for machine config pool and SR-IOV networks. 