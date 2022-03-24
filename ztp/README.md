# Cloud-Native RAN Zero Touch Provisioning

## Introduction

This folder contains a example configurations for 5G radio access network (RAN) site configurations.  Define your infrastructure as code, use declarative approaches to ensure the clusters you deploy achieve the goals you desire.  We have example configurations that can be leveraged and adapted to a mobile network operator's specific DU node configuration needs.

## RAN considerations

For RAN applications hosted on K8s clusters, very specific deployment requirements need to be met.  A declarative methodology will allow the end user to deploy the needed operators and configuration.  The end result is that the needed parameters and deployment configurations will be deployed on your cluster at the edge of the network.  

Example parameters:

* RT-kernel
* Machine config Operator
* Performance Add-on
* SRIOV Operator
* PTP Operator

## Profile configuration

We suggest breaking down the site plan into components that are common, relevant to a group of nodes and then lastly site specific details.

* Common: SCTP
* Group: PTP configuration, Performance Add-on details
* Site: IP addresses, SRIOV configuration

We look forward to user feedback and will gladly accept pull requests and issues for consideration.
