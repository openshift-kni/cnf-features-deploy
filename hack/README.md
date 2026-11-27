# hack GuideLines

This document describes how you can use the scripts from [`hack`](.) directory
and gives a bried introduction and explanation of these scripts.

## Overview

The [`hack`](.) directory contains many scripts that ensure continuous development of cnf features,
enhance the robustness of the code, improve development efficiency, etc.
The explanations and descriptions of these scripts are helpful for contributors.

## Updating the release version

Run `make update-release-version VERSION=5.1` to align OCP and operator defaults,
CNF and DPDK image tags, Dockerfile release references, the `oc` image, CNF
builder images, Git submodule branches, ZTP labels, and Tekton pipeline names
and contents.
On master, it preserves the `master` pipeline triggers and leaves unpinned
submodules unpinned. On release branches, it updates their release references.
The command also repairs partial updates. It uses the working Konflux `oc`
images from OpenShift 4.21 for release 5.0 and 4.22 for release 5.1. For
another minor release, specify both the CNF builder and an accessible `oc`
image version, for example
`make update-release-version VERSION=5.2 GO_BUILDER_VERSION=1.27 OC_IMAGE_VERSION=4.22`.

Run `make verify-release-version` to check that all managed references agree;
the check also runs as part of `make ci-job`. It knows that the CNF builder
images use Go 1.25 for release 5.0 and Go 1.26 for release 5.1. Go module
declarations, the ZTP builder image, and other dependencies remain independent.
Major release upgrades also need registry changes and are not supported by this
command.

## setup-test-cluster.sh
This script does all the cluster setup parts that are expected to be done by admins, like labelling nodes and creating the MachineConfigPool resource, etc.  
It does not install CNF.

For PTP it is possible to label nodes as non ptp capable either by  
```kubectl label <node> node-role.kubernetes.io/virtual```  
or pass a different NON_PTP_LABEL as environment variable
