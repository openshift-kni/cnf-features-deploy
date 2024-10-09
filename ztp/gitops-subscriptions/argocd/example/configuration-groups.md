# Motivation
When using hub side templating, we can have fewer PGTs to configure and manage our spokes.

Each group/site PGT can use templates for its specific configuration that can be further obtained from ConfigMaps.

Below are options for grouping source CRs into PGTs in the templating scenario.
Depending on the exact purpose and configuration for the spokes, some of the source CRs could be either common to all spokes or to different groups.

# PGT1 - configuration that can be common for most spoke clusters
**The source CRs from below could lack group or site specific info and can have common configuration for all spokes**
* AmqInstance.yaml
* ClusterLogForwarder.yaml
    * spec.outputs
    * spec.pipelines
* DisableOLMPprof.yaml
* DisableSnoNetworkDiag.yaml (Cluster Network Operator)
* HardwareEvent.yaml
* ImageRegistryConfig.yaml (Cluster Image Registry Operator)
* ImageRegistryPV.yaml
* MachineConfigGeneric.yaml
* MachineConfigPool.yaml
* MachineConfigSctp.yaml
* OperatorHub.yaml
* StorageClass.yaml
* StoragePV.yaml (Local Storage Operator)
* StoragePVC.yaml (Local Storage Operator)


# PGT2 - has configuration that can be common to sites with the same hardware(disks, NICs)/OS, mountpoints, etc
* AmqInstance.yaml
* ClusterLogForwarder.yaml
    * spec.outputs
    * spec.pipelines
* DisableSnoNetworkDiag.yaml (Cluster Network Operator)
    > Note: If users want to configure things apart spec.disableNetworkDiagnostics
* HardwareEvent.yaml
     * spec.transportHost
     * spec.logLevel
* ImageRegistryConfig.yaml (Image Registry Operator)
    > Note: Any of the spec fields can differ based on the spokes groups
* ImageRegistryPV.yaml
    > Note: Any of the spec fields can differ based on the spokes groups
* PerformanceProfile.yaml / PerformanceProfile-SetSelector.yaml 
    * spec.cpu.isolated, spec.cpu.reserved
    * spec.hugepages.pages.(size/count/node), spec.hugepages.defaultHugepagesSize
* TunedPerformancePatch.yaml
* MachineConfigGeneric.yaml
* MachineConfigPool.yaml
* MachineConfigSctp.yaml
* PtpOperatorConfigForEvent.yaml / PtpOperatorConfigForEvent-SetSelector.yam / PtpOperatorConfig.yaml
* PtpConfig<Boundary/Slave/GmWpc/Master/Slave/SlaveCvl>.yaml DU example
    * spec.profile.interface from ConfigMap
* SriovNetwork.yaml
* SriovNetworkNodePolicy.yaml / SriovNetworkNodePolicy-SetSelector.yaml
* SriovOperatorConfig.yaml
* ImageRegistryConfig.yaml (Image Registry Operator)
* ImageRegistryPV.yaml
* StorageLocalVolume.yaml
    > Note: could be common to all spokes, depending on the exact disk configuration, but safer to have it per-site (Local Storage Operator)
    * spec.devicePaths
* StorageClass.yaml
* StorageLVMCluster.yaml
* StoragePV.yaml
* StoragePVC.yaml

# PGT3 - has config that is (can be) site specific
* SriovFecClusterConfig.yaml
    * spec.acceleratorSelector.pciAddress
    * spec.physicalFunction.bbDevConfig
* SriovNetwork.yaml
* SriovNetworkNodePolicy.yaml / SriovNetworkNodePolicy-SetSelector.yaml
* StorageLocalVolume.yaml
    * spec.devicePaths
