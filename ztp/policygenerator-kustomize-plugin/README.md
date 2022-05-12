## Policy generator kustomize plugin

The policy generator kustomize plugin consumes the [policygenerator](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/policygenerator) library and [PolicyGenTemplate](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/ran-crd/policy-gen-template-crd.yaml) as a kustomize plugin. Kustomization.yaml is an example of how to use the policygenerator plugin.
```
generators:
- testPolicyGenTemplate/common-ranGen.yaml
- testPolicyGenTemplate/group-du-ranGen.yaml
- testPolicyGenTemplate/group-du-sno-ranGen.yaml
- testPolicyGenTemplate/site-du-sno-1-ranGen.yaml
```


## Build and execute
- Run the following command to build the policygenerator binary and create the kustomize plugin directory structure:
```
    $ make build
```

After the build success, the `kustomize/` directory should be created with following structure:

```
kustomize/
└── plugin
    └── ran.openshift.io
        └── v1
            └── policygentemplate
                ├── PolicyGenTemplate
                └── source-crs
                    ├── AcceleratorsNS.yaml
                    ├── AcceleratorsOperGroup.yaml
                    ├── AcceleratorsSubscription.yaml
                    ├── AmqInstance.yaml
                    ├── AmqSubscriptionNS.yaml
                    ├── AmqSubscriptionOperGroup.yaml
                    ├── AmqSubscription.yaml
                    ├── ClusterLogCatSource.yaml
                    ├── ClusterLogForwarder.yaml
                    ├── ClusterLogging.yaml
                    ├── ClusterLogNS.yaml
                    ├── ClusterLogOperGroup.yaml
                    ├── ClusterLogSubscription.yaml
                    ├── ConsoleOperatorDisable.yaml
                    ├── DefaultCatsrc.yaml
                    ├── DisableSnoNetworkDiag.yaml
                    ├── DisconnectedICSP.yaml
                    ├── MachineConfigAcceleratedStartup.yaml
                    ├── MachineConfigChronyDynamicMaster.yaml
                    ├── MachineConfigContainerMountNS.yaml
                    ├── MachineConfigPool.yaml
                    ├── MachineConfigSctp.yaml
                    ├── OperatorHub.yaml
                    ├── PerformanceProfile.yaml
                    ├── PtpCatSource.yaml
                    ├── PtpConfigMaster.yaml
                    ├── PtpConfigSlaveCvl.yaml
                    ├── PtpConfigSlave.yaml
                    ├── PtpOperatorConfigForEvent.yaml
                    ├── PtpSubscriptionNS.yaml
                    ├── PtpSubscriptionOperGroup.yaml
                    ├── PtpSubscription.yaml
                    ├── ReduceMonitoringFootprint.yaml
                    ├── SriovCatSource.yaml
                    ├── SriovNetworkNodePolicy.yaml
                    ├── SriovNetwork.yaml
                    ├── SriovOperatorConfig.yaml
                    ├── SriovSubscriptionNS.yaml
                    ├── SriovSubscriptionOperGroup.yaml
                    ├── SriovSubscription.yaml
                    ├── StorageCatSource.yaml
                    ├── StorageLV.yaml
                    ├── StorageNS.yaml
                    ├── StorageOperGroup.yaml
                    ├── StorageSubscription.yaml
                    ├── TunedPerformancePatch.yaml
                    └── validatorCRs
                        └── informDuValidator.yaml
```

Note: The source-crs directory contain the CRs (Custom Resources) that will be used to construct the ACM policies.

- Run the following command to execute kustomization.yaml
```
    $ make test
```

- Run the following command to dump the kustomization output to files under the `out/` directory
```
    $ make gen-files
```
