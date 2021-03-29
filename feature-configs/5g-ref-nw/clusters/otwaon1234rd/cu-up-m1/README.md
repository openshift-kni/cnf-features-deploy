# Data center cu-up
This is an example of a data center cu-up configurations
## Structure
The structure is identical to [du-dual-m1](../otwaon1234rd/du-dual-m1).

## Labeling
In addition to having the master role, the node must be labeled as follows:
- `node-role.kubernetes.io/worker-cu-up-otwaon1234rd=""` - used for machine config pool
- `ran.example.com/worker-cu-up-otwaon1234rd=""=""` - used as node selector for SR-IOV networks
