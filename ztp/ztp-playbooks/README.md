# Cloud-Native RAN Zero Touch Provisioning

## Introduction

This folder contains a example configurations for 5G radio access network (RAN) site configurations.  Define your infrastructure as code, use declarative approaches to ensure the clusters you deploy achieve the goals you desire.  We have example configurations that can be leveraged and adapted to a mobile network operator's specific DU node configuration needs.

## RAN considerations

For RAN applications hosted on K8s clusters, very specific deployment requirements need to be met.  A declarative methodology will allow the end user to deploy the needed operators and configuration.  You will get the needed performance parameters deployed to each node.

Example parameters:

* RT-kernel
* Machine config operator
* Performance Add-on Operator (PAO)
* SRIOV Operator
* PTP Operator

## Profile configuration

We suggest breaking down the site plan into components that are common, relevant to a group of nodes and then lastly site specific details.

* Common: SCTP
* Group: PTP configuration, Performance Add-on Operator (PAO) details
* Site: IP addresses, SRIOV configuration

We look forward to user feedback and will gladly accept pull requests and issues for consideration.
