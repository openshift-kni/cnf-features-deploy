# Workload Partitioning Feature

Workload partitioning allows the platform to dedicate specific CPUSets to customer workloads and platform workloads (~2 cores for platform and the rest for customers). This feature **must** be activated at install time of a cluster, once active that cluster is locked into functioning with this feature on.

## Part 1 - Configure PolicyGenTemplate
- This will pin all the host level services such as systemd and crio.
- Update PerformanceProfile in `PolicyGenTemplate`. See example [here](https://github.com/openshift-kni/cnf-features-deploy/blob/82ff3617a5e69b47b1f8d8b5d4a8db7719ab4bb4/ztp/gitops-subscriptions/argocd/example/policygentemplates/group-du-sno-ranGen.yaml#L99).
- In this context we will look at [`reserved`](https://github.com/openshift-kni/cnf-features-deploy/blob/82ff3617a5e69b47b1f8d8b5d4a8db7719ab4bb4/ztp/gitops-subscriptions/argocd/example/policygentemplates/group-du-sno-ranGen.yaml#L105)
  - The value here is of CPU core numbers (e.g `0-3`) and is dependent on the hardware.
- You can put the values as you see fit. But you can use "Performance Profile Creator" [tool](https://docs.openshift.com/container-platform/4.11/scalability_and_performance/cnf-create-performance-profiles.html#cnf-about-the-profile-creator-tool_cnf-create-performance-profiles) to assist with selecting cores in `reserved` to maximize the benefit and ensure correctness (see WARNING). 
  - look for `--reserved-cpu-count` when using Performance Profile Creator cli.
- WARNING: PerformanceProfile's `reserved` and `isolated` must span ALL AVAILABLE CORES and not doing so will result in an undefined behaviour (generating it with tool above should take care of this behind-the-scenes)

## Part 2 - Configure SiteConfig
- This will pin all the platform applications such as ovn and apiserver
- Configure workload partitioning in `SiteConfig`, See example [here](https://github.com/openshift-kni/cnf-features-deploy/blob/82ff3617a5e69b47b1f8d8b5d4a8db7719ab4bb4/ztp/gitops-subscriptions/argocd/example/siteconfig/example-sno.yaml#L59)
- Set the value of `cpuset` must be the exact one used for `reserved` in `Part 1`

## Part 3 - Apply configured SiteConfig and PolicyGenTemplate with GitOps
Note that PerformanceProfile is configured first (`Part 1`) for correctness and convenience, but it's expected to be applied as part of day-2 operations. To maximize core reduction benefits, both (SC and PGT) must be configured and applied eventually. 
1. Apply SiteConfig
2. Apply PolicyGenTemplate

## Verification
ssh into the node (e.g `oc debug node/<NODE_NAME>` followed by `chroot /host`) and try: 
- Look for cpu pinning with `taskset`
  1. pinning done with PerformanceProfile and output must match `reserved`
     - `pgrep "systemd|crio|kubelet" | while read i; do echo "CPUSet $(taskset -cp $i | grep -Po '[0-9]+[-,]+[0-9]+.*') for process $(ps -p $i -o comm=)"; done`
  2. pinning done with Workload Partitioning and output must match `cpuset`
     - `pgrep "ovn|apiserver" | while read i; do echo "CPUSet $(taskset -cp $i | grep -Po '[0-9]+[-,]+[0-9]+.*') for process $(ps -p $i -o comm=)"; done`
  3.  Values after `..current affinity list:` for both `systemd` and `ovn` must the match
- Look for the correct config under `cat /proc/cmdline` updated with new info such as `systemd.cpu_affinity=xx`. `xx` should match the value in `reserved` from `PolicyGenTemplate`
- Look for the correct config under `cat /etc/crio/crio.conf.d/01-workload-partitioning` and make sure it matches `cpuset` in SiteConfig.

## Additional readings
- [Management Workload Partitioning](https://github.com/openshift/enhancements/blob/5c92a52b27580c96eaf7ea3af79fef35463b3e2a/enhancements/workload-partitioning/management-workload-partitioning.md)
- [Performance Profile](https://docs.openshift.com/container-platform/4.11/scalability_and_performance/cnf-low-latency-tuning.html)
- [Workload Partitioning in SNO](https://docs.openshift.com/container-platform/4.11/scalability_and_performance/sno-du-enabling-workload-partitioning-on-single-node-openshift.html)