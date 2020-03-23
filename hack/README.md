# hack GuideLines

This document describes how you can use the scripts from [`hack`](.) directory
and gives a bried introduction and explanation of these scripts.

## Overview

The [`hack`](.) directory contains many scripts that ensure continuous development of cnf features,
enhance the robustness of the code, improve development efficiency, etc.
The explanations and descriptions of these scripts are helpful for contributors.

## setup-test-cluster.sh
This script does all the cluster setup parts that are expected to be done by admins, like labelling nodes and creating the MachineConfigPool resource, etc.  
It does not install CNF.

For PTP it is possible to label nodes as non ptp capable either by  
```kubectl label <node> node-role.kubernetes.io/virtual```  
or pass a different NON_PTP_LABEL as environment variable
