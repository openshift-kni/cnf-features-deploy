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

The [`ran-profile`](ran-profile) directory contains the PolicyGenTemplates for deployment of DU integration features for three types of clusters, those PolicyGenTemplates are the symbolic links from the [`recommended DU profile`](../../ztp/gitops-subscriptions/argocd/example/policygentemplates/), that are:
- A single `common-ranGen.yaml` that should apply to all types of sites
- A set of shared `group-du-*-ranGen.yaml`, each of which should be common across a set of similar clusters
- An example `example-*-site.yaml` which will normally be copied and updated for each individual site

## Deployment

From the the project root `cnf-features-deploy`, generate the RAN profile based on the PolicyGentemplates:
```bash
linked_pgts=feature-configs/cn-ran-overlays/ran-profile/policygentemplates
source_pgts=ztp/gitops-subscriptions/argocd/example/policygentemplates

podman run --rm \
-v "$(pwd)/$linked_pgts":/resources/$linked_pgts:Z \
-v "$(pwd)/$source_pgts":/resources/$source_pgts:Z \
quay.io/openshift-kni/ztp-site-generator \
generator config -N $linked_pgts $linked_pgts/cluster-config
```
The generated artifacts will be outputted to ./feature-configs/cn-ran-overlays/ran-profile/cluster-config. It can be applied manually with oc command or with the toolset of your choice (E.g. ArgoCD)

This project contains makefile based tooling, that can be used as follows (from the project root):
```bash
  FEATURES_ENVIRONMENT=cn-ran-overlays FEATURES=ran-profile CLUSTER_TYPE=standard make generate-ran-artifacts feature-deploy
  FEATURES_ENVIRONMENT=cn-ran-overlays FEATURES=ran-profile CLUSTER_TYPE=sno make generate-ran-artifacts feature-deploy
  FEATURES_ENVIRONMENT=cn-ran-overlays FEATURES=ran-profile CLUSTER_TYPE=3node make generate-ran-artifacts feature-deploy
```

## SR-IOV configuration notes
SriovNetworkNodePolicy object must be configured differently for different NIC models and placements. 

| Manufacturer | deviceType | isRdma |
| --- | --- | --- |
| Intel | __vfio-pci__ or __netdevice__ | __false__ |
| Mellanox | __netdevice__ | __true__ |


In addition, when configuring the `nicSelector`, `pfNames` value must match the intended interface name on the specific host.

If there is a mixed cluster where some of the nodes are deployed with Intel NICs and some with Mellanox, several SR-IOV configurations can be created with the same `resourceName`. The device plugin will discover only the available ones and will put the capacity on the node accordingly.