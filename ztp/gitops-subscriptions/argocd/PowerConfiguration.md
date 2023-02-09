# Power Configuration

Low latency, high performance edge deployments require C-states and P-states to be disabled or limited. This may result in each cpu running at the same maximum turbo frequency that is supported for all processors on that particular server. This provides the best latency for workloads, but also consumes the most power. However, not all workloads require such power intensive configurations. Workloads can be categorized as either *critical* or *non-critical*. Critical workloads require disabled or limited C-state and P-state settings for high performance and low latency, whereas non-critical workloads use C-state and P-state settings for power savings at the cost of some latency/performance for those non-critical workloads.

This document provides deployment notes on how to use ZTP to configure power modes for performance (low latency), high performance (ultra-low latency), power saving and a mix of power saving and performance (i.e. low latency) for workloads on the same OpenShift server (using per-pod C-state/P-state configuration).

Power configurations are mainly set using the `workloadHints` object in the PerformanceProfile. Detailed descriptions of each parameter can be found in the Node Tuning Operator [PerformanceProfile documentation](https://github.com/openshift/cluster-node-tuning-operator/blob/master/docs/performanceprofile/performance_profile.md#workloadhints).

## Prerequisite

In order to use the power saving mode, it is required that C-states and OS-controlled P-states are enabled through BIOS/firmware settings. Different vendors use varying naming for their BIOS settings, but here are some guidelines:

- enable C-states:
    - often referred to as “monitor/mwait” instructions or “C states” (or both)
    - there may also be options to enable/disable specific C-states (e.g. C1E, C6)
    - users may want to enable all C-states or just a subset
- enable OS-controlled P-states:
    - this can be called “CPU power management” or “hardware P-state control”
    - the key is to choose the option(s) that allow the OS to control P-states

## Power modes

### 1) [Performance mode](#performance-mode) (low latency) \[default\]

Deploy the PerformanceProfile and TunedPerformancePatch policies with their default configurations.

### 2) [High performance mode](#high-performance-mode) (ultra-low latency)

Deploy the PerformanceProfile and TunedPerformancePatch policies as above with the addition of a new configuration under the `workloadHints` specification with `highPowerConsumption` enabled.

```yaml
  sourceFiles:
    - fileName: PerformanceProfile.yaml
      policyName: "config-policy"
      metadata:
        name: openshift-node-performance-profile
      spec:
        ...
        workloadHints:
          realTime: true
          highPowerConsumption: true
        ...
```

### 3) [Power saving mode](#power-saving-mode)

Deploy the PerformanceProfile and TunedPerformancePatch policies as previously with the following changes to the `workloadHints` specification. Set `highPowerConsumption` to `false` and `perPodPowerManagement` to `true`. It is important to note that both fields cannot be simultaneously enabled as this will result in an error. Furthermore, an additional kernel argument for the cpu governor can also be configured. The `schedutil` governor is recommended, however, other governors that can be used include `ondemand` and `powersave`.

In order to maximize the power saving gain, the maximum cpu frequency should be capped. Without limiting the maximum cpu frequency, enabling C-states on the non-critical workload cpus allows the frequency of the critical cpus to be boosted, negating much of the power savings.

The tuned performance-patch can be used to confine the maximum cpu frequency. The profile section of the performance-patch can be extended to use the `sysfs` plugin to set the `max_perf_pct`, which applies to all cpus. The `max_perf_pct` parameter controls the maximum frequency the cpufreq driver is allowed to set as a percent of the maximum supported cpu frequency (/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo\_max\_freq). This is demonstrated in the example below.

> **Note** `/sys/devices/system/cpu/intel_pstate/max_perf_pct=x` where `x` is to be configured based on the hardware specification and powersaving requirements.

```yaml
  sourceFiles:
    - fileName: PerformanceProfile.yaml
      policyName: "config-policy"
      metadata:
        name: openshift-node-performance-profile
      spec:
        # Use the additionalKernelArgs list as defined in ztp/source-crs/PerformanceProfile.yaml
        additionalKernelArgs:
          - "cpufreq.default_governor=schedutil"
          - "rcupdate.rcu_normal_after_boot=0"
          - "efi=runtime"
        ...
        workloadHints:
          realTime: true
          highPowerConsumption: false
          perPodPowerManagement: true
        ...
    - fileName: TunedPerformancePatch.yaml
      policyName: "config-policy"
      spec:
        profile:
          - name: performance-patch
            data: |
              ...
              [sysfs]
              /sys/devices/system/cpu/intel_pstate/max_perf_pct=72
```

More information pertaining to the Power saving configuration can be found in the official [OpenShift docs](https://docs.openshift.com/container-platform/4.12/scalability_and_performance/cnf-low-latency-tuning.html#node-tuning-operator-pod-power-saving-config_cnf-master).

####  [Optional: Power saving mode](#optional-power-saving-mode)

For critical workloads that require the highest performance and lowest latency, each such pod can be annotated to disable C-states and the cpufreq governor set to `performance` as shown below:

```yaml
metadata:
  annotations:
    cpu-c-states.crio.io: "disable"
    cpu-freq-governor.crio.io: "performance"
```

It is worthwhile to note the following:

- These annotations are likely to be used with the other high performance cri-o annotation described [here](https://docs.openshift.com/container-platform/4.12/scalability_and_performance/cnf-low-latency-tuning.html#node-tuning-operator-pod-power-saving-config_cnf-master).
- These annotations have the same restrictions as the other high performance cri-o annotations:
    - The pod must use the performance-&lt;profile-name&gt; runtime class.
    - The pod must have a [QoS Class of guaranteed](https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/#create-a-pod-that-gets-assigned-a-qos-class-of-guaranteed) (i.e. uses whole CPUs).

## Migrating from one mode to another

This section outlines the steps and configurations for migrating from one mode to another mode.

### 1) Performance to Power saving

Migrating from a Performance configuration to a Power saving configuration requires setting the workloadHints parameters as follows:

```yaml
workloadHints:
  realTime: true
  highPowerConsumption: false
  perPodPowerManagement: true
```

> **Note**: Critical workload pods should be annotated with the [crio annotations](#optional-power-saving-mode) shown above prior to the migration.

It is recommended to set a cpu governor in the additional kernel arguments specification as well as capping the max cpu frequency as explained in the [power saving mode](#3-power-saving-mode).

When using the recommended GitOps and Policy solution for managing cluster configuration, the above changes need to be propagated to the cluster(s) to take effect. Note that any changes to the PerformanceProfile and/or kernel arguments will result in a reboot of the affected node(s).

After the changes have been applied to the affected SNOs, the change can be verified by viewing the PerformanceProfile on the SNOs:

```sh
oc get PerformanceProfile -o yaml
```

### 2) Power saving to Performance

Migrating from a Power saving configuration to a [Performance (low latency)](#1-performance-mode-low-latency-default) configuration requires setting the workloadHints parameters as follows:

```yaml
workloadHints:
  realTime: true
  highPowerConsumption: false
  perPodPowerManagement: false
```

If a cpu governor is configured via the additional kernel arguments, and the configuration is managed through Policy, this must be removed according to the `complianceType` of the policy. If the `complianceType` is configured to `mustonlyhave`, then this can be accomplished by setting the `additionalKernelArgs` to the original list as defined in the `ztp/source-crs/PerformanceProfile.yaml` file. This is shown below:

```yaml
additionalKernelArgs:
  - "cpufreq.default_governor=schedutil"
  - "rcupdate.rcu_normal_after_boot=0"
```

If however the `complianceType` is configured to `musthave`, then the additional kernel argument must be removed as described in [NTO configuration hot fixes](https://github.com/openshift/cluster-node-tuning-operator/blob/master/docs/performanceprofile/configuration_hotfixes.md#additional-kernel-arguments).

> **Note** that it is recommended to ensure that the `sysfs` plugin (responsible for capping the max cpu frequency) is not present in the TunedPerformancePatch profile.

When using the recommended GitOps and Policy solution for managing cluster configuration, the above changes need to be propagated to the cluster(s) to take effect. Note that any changes to the PerformanceProfile and/or kernel arguments will result in a reboot of the affected node(s).

After the policy has been applied to the affected SNOs, the change can be verified by viewing the PerformanceProfile on the SNOs:

```sh
oc get PerformanceProfile -o yaml
```

## Validating the Power configuration settings

In order to validate whether the correct power configuration mode has been applied via ZTP post deployment, the `/proc/cmdline` config on the nodes can be verified as shown below.

> **Note**: If power saving is enabled, `intel_pstate` is set to `passive`, otherwise for performance modes it is set to `disable`.

```sh
sh-4.4# cat /proc/cmdline
... intel_pstate=passive cpufreq.default_governor=schedutil
```

To verify that the Power saving mode is correctly applied on the SNO, the following items can be checked:

- cpu governor (if configured via `additionalKernelArgs`) can be seen in the output of `cat /proc/cmdline` above. It can also be confirmed as follows:

```sh
sh-4.4# cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_driver
intel_cpufreq (NOTE: this will be the same for all cpus)

sh-4.4# cat /sys/devices/system/cpu/cpuidle/current_driver
intel_idle
```

- max cpu frequency (if configured via TunedPerformancePatch policy):

```sh
sh-4.4# cat /sys/devices/system/cpu/intel_pstate/max_perf_pct
<x> (NOTE: this should match the % that was configured)
```
