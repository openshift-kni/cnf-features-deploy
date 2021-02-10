# Description

Based on this [proof-of-concept](https://github.com/lack/redhat-notes/tree/main/crio_unshare_mounts)

Creates a pair of MC objects, one for masters and one for workers, which do the same thing in each case:
- Create a new 'container-mount-namespace.service' which manages a unique mount namespace for CRI-O and Kubelet
- Creates systemd drop-ins for CRI-O and Kubelet so they execute within this mount namespace

The goal is to segregate all contianer-specific mountpoints from systemd to reduce systemd CPU usage.
