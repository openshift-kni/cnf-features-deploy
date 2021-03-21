# Profile base
This folder contains the basic building blocks all specific clusters are based on.
  - [ran-general](./ran-general) contains everything common for all RAN nodes:
      - sctp
      - TODO: Add xt_u32
  - [du-general](./du-general) contains configuration parts common for all DUs and not covered in [ran-general](./ran-general):
      - ptp
  - [du-dual](./du-dual) contains configuration parts specific to the `du-dual` role-hardware flavor implemented on a dual-socket server. This includes
      - performance profile
      - SR-IOV network node policies and network configurations
  - [cu-up](./cu-up) contains configuration parts specific to cu-up flavor. This includes
      - performance profile
      - SR-IOV network node policies and network configurations
  - [du-single](./du-single) contains configuration parts specific to the `du-single` role-hardware flavor implemented on a dual-socket server. The content is similar to the [du-dual](./du-dual). The differences are in the smaller amount of network definitions and smaller amount of CPU cores.

The profiles described above don't refer to a specific hardware, distinguishing only between single-socket and dual-socket servers. In practice, however, the hardware and its settings will be specific. The customers and partners are expected to integrate their solutions on a finite set of hardware configurations. These specific configurations are currently out of scope of this work. In the future, certified role-hardware flavors can be placed in the `certified-platforms` folder. They can be built from scratch, or by overlaying the basic building blocks, such as `du-dual`, `du-single` etc.


## <a name="hw_types"></a>Assumptions on hardware types
This chapter contains hardware specification examples. These are not recommendations and are only provided as a reference to the correspondent performance and network profiles in this repository.
 

### Single-socket servers <a name="du_single"></a>
Single-socket server are targeted RAN DU on single node, remote worker node and three-node cluster deployments.


#### Hardware details
|  |  |
| --- | --- |
| CPU | Intel Xeon Gold series |
| RAM | 96GB, DDR4 |
| Disks | 1TB /dev/nvme0n1 |
| NUMA 0 | CPUs:<br>0 - 51 (with HT)<br>0 - 26 (without HT) |


#### PCI cards
| Device name | Model | Comment |
| --- | --- | --- |
| eno1 | 10GB-T | On-board NIC, not used|
| eno2 | 10GB-T | On-board NIC, not used|
| ens1f0 | XXV710 | Midhaul networks |
| ens1f1 | XXV710 | Not used for traffic |
| ens3f0 | N3000 | Fronthaul, PTP |
| ens3f1 | N3000| Not used for traffic |
| n/a | N3000| Accelerator |

###  Dual-socket servers - CU <a name="cu_hw"></a>
Used for CU-CP, cu-up and 5GC nodes in data center deployments.
There might be a difference between CU-CP and cu-up network configuration, but there seem to be no differences between the two with respect to the compute requirements.

#### Hardware details
|  |  |
| --- | --- |
| CPU | Intel Xeon Gold series |
| RAM | 192GB, DDR4 |
| Disks | 480GB /dev/ssda <br> 6 x 3.2TB NVMe |
| NUMA 0 | CPUs:<br>0, 2, 4, - 102 EVEN (with HT)<br>0, 2, 4, - 50 EVEN (without HT) |
| NUMA 0 | CPUs:<br>1, 3, 5, - 103 ODD (with HT)<br>1, 3, 5, - 51 ODD(without HT) |

#### PCI cards
| Device name | Model | Comment |
| --- | --- | --- |
| eno1 | 10GB-T | On-board NIC, not used|
| eno2 | 10GB-T | On-board NIC, not used|
| ens1f0 | XXV710 | Midhaul networks, NUMA 0 |
| ens1f1 | XXV710 | Backhaul networks, NUMA 0 |
| ens3f0 | XXV710 | Midhaul networks, NUMA 1 |
| ens3f1 | XXV710 | Backhaul networks, NUMA 1 |

### Dual-socket servers - DU  <a name="du_dual"></a>
#### Hardware details
Same as [Dual-socket servers - CU](#cu_hw)
#### PCI cards
| Device name | Model | Comment |
| --- | --- | --- |
| eno1 | 10GB-T | On-board NIC, not used|
| eno2 | 10GB-T | On-board NIC, not used|
| ens1f0 | XXV710 | Midhaul networks, NUMA 0 |
| ens1f1 | XXV710 | Not used |
| ens3f0 | XXV710 | Midhaul networks, NUMA 1 |
| ens3f1 | XXV710 | Backhaul networks, NUMA 1 |
| ens5f0 | N3000 | Fronthaul networks, NUMA 0 |
| ens5f1 | N3000 | Not used |
| ens7f0 | N3000 | Fronthaul networks, NUMA 1 |
| ens7f1 | N3000 | Not used |
| | N3000 | Accelerator NUMA 0|
| | N3000 | Accelerator NUMA 1|

