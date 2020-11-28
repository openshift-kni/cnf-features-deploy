# Cloud-native RAN profile

## Introduction
This folder contains an example configurations for 5G radio access network sites.
Radio access network is composed of Centrallized units (CU), distributed units (DU) and Radio units (RU).
RAN from the telecommunications standard perspective is shown below:

<img src="images/ran.png">

From the three components composing RAN, only CU and DU can be virtualized and implemented as cloud-native functions.
CU and DU split is driven by real-time computing and networking requirements. A DU can be seen as a real-time part of a telecommunication baseband unit. One distributed unit may aggregate several cells. A CU can be seen as a non-realtime part of a baseband unit, aggregating traffic from one or more distributed units.

A cell in the context of a DU can be seen as a real-time application performing intensive digital signal processing, data transfer and algorithmic tasks. Cells are often using hardware acceleration (FPGA, GPU, eASIC) for DSP processing offload, but there are also software-only implementations (FlexRAN), based on AVX-512 instructions. 
Running cell application on COTS hardware requires following features to be enabled:

- Real-time kernel
- CPU isolation
- NUMA awareness
- HUGEPAGES memory management
- Precision timing synchronization using PTP
- AVX-512 instruction set (for Flexran and / or FPGA implementation)
- Additional features depending on the RAN operator requirements

Accessing hardware acceleration devices and high throughput network interface cards by virtualized software applications requires use of SRIOV and Passthrough PCI device virtualization.
In addition to the compute and acceleration requirements, DUs operate on multiple internal and external networks.

## Overview

Current example is focused on a DU. A CU profile can be built by
- Replacing PTP by NTP (Work in progress)
- Using stock kernel instead of RT kernel
- Skipping FEC (Work in progress)

The [`ran-profile`](ran-profile) directory contains the Kustomize profile for deployment of DU integration features, namely:
- SCTP MachineConfig patch
- Performance addon operator and DU performance profile
- PTP operator and slave profile
- SR-IOV operator and associated profiles

## The manifest structure

The profile is built from one cluster specific folder and one or more site-specific folders. This is done to address a deployment that includes remote worker nodes (several sites belonging to the same cluster).
The [`cluster-config`](ran-profile/cluster-config) directory contains performance and PTP customizations based upon operator deployments in [`deploy`](../feature-configs/deploy) folder.
The [`site.1.fqdn`](site.1.fqdn) folder contains site-specific network customizations.


## Prerequisites

1. Create a machine config pool for the RAN worker nodes. For example:

```
cat <<EOF | oc apply -f -
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfigPool
metadata:
  name: worker-cnf
  labels:
    machineconfiguration.openshift.io/role: worker-cnf
spec:
  machineConfigSelector:
    matchExpressions:
      - {
          key: machineconfiguration.openshift.io/role,
          operator: In,
          values: [worker-cnf, worker],
        }
  paused: false
  nodeSelector:
    matchLabels:
      node-role.kubernetes.io/worker-cnf: ""
---
EOF
```

2. Include the worker node in the above machine config pool by labelling it with `node-role.kubernetes.io/worker-cnf` label:

```bash
oc label --overwrite node/{your node name} node-role.kubernetes.io/worker-cnf=""
```
3. Label the node as ptp slave (DU only):

```bash
oc label --overwrite node/{your node name} ptp/slave=""
```

An example of labelling the nodes used in CI/CD can be seen in `cnf-features-deploy/hack/setup-test-cluster.sh`


## Deployment

The profile is built in layers with __kustomize__.
To get the profile output, run 
```bash
oc kustomize ran-profile
```
It can be applied manually or with the toolset of your choice (E.g. ArgoCD)

This project contains makefile based tooling, that can be used as follows (from the project root):

  `FEATURES_ENVIRONMENT=cn-ran-overlays FEATURES=ran-profile make feature-deploy`

## SR-IOV configuration notes
SriovNetworkNodePolicy object must be configured differently for different NIC models and placements. 

| Manufacturer | deviceType | isRdma |
| --- | --- | --- |
| Intel | __vfio-pci__ or __netdevice__ | __false__ |
| Mellanox | __netdevice__ | __true__ |


In addition, when configuring the `nicSelector`, `pfNames` value must match the intended interface name on the specific host.

If there is a mixed cluster where some of the nodes are deployed with Intel NICs and some with Mellanox, several SR-IOV configurations can be created with the same `resourceName`. The device plugin will discover only the available ones and will put the capacity on the node accordingly.