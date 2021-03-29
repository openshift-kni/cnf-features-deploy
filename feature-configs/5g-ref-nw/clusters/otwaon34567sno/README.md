# Single node cluster
This is an example of a Single node cluster DU
## Structure
The structure is identical to [du-dual-m1](../otwaon1234rd/du-dual-m1).
The only difference here - we don't configure a separate machine config pool for the role-hardware pool, because in the single node cluster `master` MCP is used for everything.

## Labeling
In addition to having the master role, the node must be labeled as follows:
- `ptp/slave=""` - used as node selector for PTP
- `ran.example.com/worker-du-single-otwaon34567sno=""` - used as node selector for SR-IOV networks

## Log forwarding
While standard clusters use local log storage, in case of a single node cluster we assume the logs to be forwarded to an external log aggregating service.
This profile contains a cluster configuration example for log forwarding to an external Kafka bus:
[Kafka](./nw-config/logging/README.md)
