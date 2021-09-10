# PolicyGen (Policy Generator) & SiteConfig (SiteConfig Generator)

## 1. PolicyGen
PolicyGen is a plugin used to facilitate creating ACM policies from a predefined Custom Resources (CRs). There are 3 main items (Policy Categorization, Source CR policy and PolicyGenTemplate) that PolicyGen rely on to generate the ACM policies and its placement binding and placement rule.

1 - Policy Categorization:

For a large scale deployment, the ACM policies that will be applied to the OpenShift clusters need to be categorized. PolicyGen categorize the ACM policies as follow;

   - Common (policies): policies that will be applied to all managed OpenShift clusters.

   - Group (policies): policies that will be applied for a group of the managed OpenShift clusters.

   - Sites/clusters (policies): policies that will be applied to a single managed OpenShift cluster.

2 - Source CR policy:

The source CR that will be used to generate the ACM policy needs to be defined with consideration of possible overlay to its metadata or spec/data. For example; a common-namespace-policy contain a Namespace CR that will created/applied on all managed clusters. This namespace will be placed under the common category and there will be no changes for its spec or data across all clusters. The source CR for this namespace could be as below

```
apiVersion: v1
kind: Namespace
metadata:
 name: openshift-sriov-network-operator
 labels:
   openshift.io/run-level: "1"
```

The generated policy that will apply this namespace will include the namespace as it is defined above without any change. It will be as below 

```
apiVersion: policy.open-cluster-management.io/v1
kind: Policy
metadata:
   name: common-sriov-sub-ns-policy
   namespace: common-sub
   annotations:
       policy.open-cluster-management.io/categories: CM Configuration Management
       policy.open-cluster-management.io/controls: CM-2 Baseline Configuration
       policy.open-cluster-management.io/standards: NIST SP 800-53
spec:
   remediationAction: enforce
   disabled: false
   policy-templates:
       - objectDefinition:
           apiVersion: policy.open-cluster-management.io/v1
           kind: ConfigurationPolicy
           metadata:
               name: common-sriov-sub-ns-policy-config
           spec:
               remediationAction: enforce
               severity: low
               namespaceselector:
                   exclude:
                       - kube-*
                   include:
                       - '*'
               object-templates:
                   - complianceType: mustonlyhave
                     objectDefinition:
                       apiVersion: v1
                       kind: Namespace
                       metadata:
                           labels:
                               openshift.io/run-level: "1"
                           name: openshift-sriov-network-operator
```

Another example; a SriovNetworkNodePolicy CR that will be exist in different OpenShift clusters with different spec for each cluster. The source CR for the [SriovNetworkNodePolicy](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/source-crs/SriovNetworkNodePolicy.yaml) as below.

```
apiVersion: sriovnetwork.openshift.io/v1
kind: SriovNetworkNodePolicy
metadata:
  name: sriov-nnp
  namespace: openshift-sriov-network-operator
spec:
  # The $ tells the policy generator to overlay/remove the spec.item in the generated policy.
  deviceType: $deviceType
  isRdma: false
  nicSelector:
    pfNames: [$pfNames]
  nodeSelector:
    node-role.kubernetes.io/worker: ""
  numVfs: $numVfs
  priority: $priority
  resourceName: $resourceName
```

The SriovNetworkNodePolicy name and namespace will be same for all clusters so both are defined in the source SriovNetworkNodePolicy. However, the generated policy will required the $deviceType, $numVfs,...etc as input parameters in order to adjust the policy for each cluster. The generated policy will be as below

```
apiVersion: policy.open-cluster-management.io/v1
kind: Policy
metadata:
    name: site-du-sno-1-sriov-nnp-mh-policy
    namespace: sites-sub
    annotations:
        policy.open-cluster-management.io/categories: CM Configuration Management
        policy.open-cluster-management.io/controls: CM-2 Baseline Configuration
        policy.open-cluster-management.io/standards: NIST SP 800-53
spec:
    remediationAction: enforce
    disabled: false
    policy-templates:
        - objectDefinition:
            apiVersion: policy.open-cluster-management.io/v1
            kind: ConfigurationPolicy
            metadata:
                name: site-du-sno-1-sriov-nnp-mh-policy-config
            spec:
                remediationAction: enforce
                severity: low
                namespaceselector:
                    exclude:
                        - kube-*
                    include:
                        - '*'
                object-templates:
                    - complianceType: musthave
                      objectDefinition:
                        apiVersion: sriovnetwork.openshift.io/v1
                        kind: SriovNetworkNodePolicy
                        metadata:
                            name: sriov-nnp-du-mh
                            namespace: openshift-sriov-network-operator
                        spec:
                            deviceType: vfio-pci
                            isRdma: false
                            nicSelector:
                                pfNames:
                                    - ens7f0
                            nodeSelector:
                                node-role.kubernetes.io/worker: ""
                            numVfs: 8
                            resourceName: du_mh
```
Note: Define the required input parameters as $value (ex: $deviceType) is not mandatory. The $ tells the policy generator to overlay/remove this item from the generated policy. Otherwise the value will stay as it is.

3 - PolicyGenTemplate:

[PolicyGenTemplate](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/ran-crd/policy-gen-template-crd.yaml) is a Custom Resource Definition (CRD) that tells PolicyGen where to categorize the generated policies and which spec/data items need to be overlaid. Let's consider the [group-du-ranGen.yaml](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/ran-crd/policy-gen-template-ex.yaml) examples to explain more.

```
# Example for using the PolicyGenTemplate to create ACM policies with binding rules group-du-sno.
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
 name: "group-du-sno"
 namespace: "group-du-sno"
spec:
  bindingRules:
    group-du-sno: ""
  mcp: "master"
  sourceFiles:
    - fileName: ConsoleOperatorDisable.yaml
      policyName: "console-policy"
    - fileName: ClusterLogging.yaml
      policyName: "cluster-log-policy"
      spec:
        curation:
          curator:
            schedule: "30 3 * * *"
        collection:
          logs:
            type: "fluentd"
            fluentd: {}
```

The previous example defines group of policies named group-du-sno. It defines 2 ACM policies; 1- console-policy which has its source CR file defined at [ConsoleOperatorDisable.yaml](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/source-crs/ConsoleOperatorDisable.yaml). The policy will disable the console UI for all OpenShift managed clusters belong to group-du-sno. 2- cluster-log-policy which has its source CR file defined at [ClusterLogging.yaml](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/source-crs/ClusterLogging.yaml). The policy will apply the spec.curation.schedule as its defined in the PolicyGenTemplate to all OpenShift managed clusters belong to group-du-sno. The generated console-policy will be as below.

```
apiVersion: policy.open-cluster-management.io/v1
kind: Policy
metadata:
  annotations:
    policy.open-cluster-management.io/categories: CM Configuration Management
    policy.open-cluster-management.io/controls: CM-2 Baseline Configuration
    policy.open-cluster-management.io/standards: NIST SP 800-53
  name: group-du-sno-console-policy
  namespace: group-du-sno
spec:
  disabled: false
  policy-templates:
    - objectDefinition:
        apiVersion: policy.open-cluster-management.io/v1
        kind: ConfigurationPolicy
        metadata:
          name: group-du-sno-console-policy-config
        spec:
          namespaceselector:
            exclude:
              - kube-*
            include:
              - '*'
          object-templates:
            - complianceType: mustonlyhave
              objectDefinition:
                apiVersion: operator.openshift.io/v1
                kind: Console
                metadata:
                  name: cluster
                spec:
                  logLevel: Normal
                  managementState: Removed
                  operatorLogLevel: Normal
          remediationAction: enforce
          severity: low
  remediationAction: enforce
```

## 2- SiteConfig generator

In order to provision an OpenShift cluster using ACM, the following CRs need to be defined; AgentClusterInstall, ClusterDeployment, NMStateConfig, KlusterletAddonConfig, ManagedCluster, InfraEnv, BareMetalHost and extra-manifest configurations (ConfigMap) that will be applied to the cluster based on the deployment use-case. [SiteConfig](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/ran-crd/site-config-crd.yaml) is Custom Resource Definition (CRD) that gather all the required configuration from the previous CRs into one SiteConfig CR. SiteConfig generator uses the siteconfig CR to generate the previously mentioned CRs to provision OpenShift cluster. As an example the siteConfig [site2-sno-du](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/ztp-policy-generator/testSiteConfig/site2-sno-du.yaml) will generate the CRs as below.
```
apiVersion: v1
kind: Namespace
metadata:
    labels:
        name: site-sno-du-2
    name: site-sno-du-2
---
apiVersion: extensions.hive.openshift.io/v1beta1
kind: AgentClusterInstall
metadata:
    name: site-sno-du-2
    namespace: site-sno-du-2
spec:
    clusterDeploymentRef:
        name: site-sno-du-2
    imageSetRef:
        name: openshift-v4.8.0
    manifestsConfigMapRef:
        name: site-sno-du-2
    networking:
        clusterNetwork:
            - cidr: 10.128.0.0/14
              hostPrefix: 23
        machineNetwork:
            - cidr: 10.16.231.0/24
        serviceNetwork:
            - 172.30.0.0/16
    provisionRequirements:
        controlPlaneAgents: 1
    sshPublicKey: 'ssh-rsa '
---
apiVersion: hive.openshift.io/v1
kind: ClusterDeployment
metadata:
    name: site-sno-du-2
    namespace: site-sno-du-2
spec:
    baseDomain: example.com
    clusterInstallRef:
        group: extensions.hive.openshift.io
        kind: AgentClusterInstall
        name: site-sno-du-2
        version: v1beta1
    clusterName: site-sno-du-2
    installed: false
    platform:
        agentBareMetal:
            agentSelector:
                matchLabels:
                    cluster-name: site-sno-du-2
    pullSecretRef:
        name: pullSecretName
---
apiVersion: agent-install.openshift.io/v1beta1
kind: NMStateConfig
metadata:
    labels:
        nmstate-label: site-sno-du-2
    name: site-sno-du-2
    namespace: site-sno-du-2
spec:
    config:
        dns-resolver:
            config:
                server:
                    - 2620:52:0:10e7:e42:a1ff:fe8a:800
        interfaces:
            - ipv6:
                address:
                    - 2620:52:0:10e7:e42:a1ff:fe8a:601/64
                    - 2620:52:0:10e7:e42:a1ff:fe8a:602/64
                    - 2620:52:0:10e7:e42:a1ff:fe8a:603/64
                dhcp: false
                enabled: true
              macAddress: "00:00:00:01:20:30"
              name: eno1
              type: ethernet
        routes:
            config:
                - destination: 0.0.0.0/0
                  next-hop-address: 2620:52:0:10e7:e42:a1ff:fe8a:999
                  next-hop-interface: eno1
                  table-id: 254
    interfaces:
        - name: eno1
          macAddress: "00:00:00:01:20:30"
---
apiVersion: agent.open-cluster-management.io/v1
kind: KlusterletAddonConfig
metadata:
    name: site-sno-du-2
    namespace: site-sno-du-2
spec:
    applicationManager:
        enabled: true
    certPolicyController:
        enabled: false
    clusterLabels:
        cloud: auto-detect
        vendor: auto-detect
    clusterName: site-sno-du-2
    clusterNamespace: site-sno-du-2
    iamPolicyController:
        enabled: false
    policyController:
        enabled: true
    searchCollector:
        enabled: false
---
apiVersion: cluster.open-cluster-management.io/v1
kind: ManagedCluster
metadata:
    labels:
        common: "true"
        group-du-sno: ""
        sites: site-sno-du-2
    name: site-sno-du-2
spec:
    hubAcceptsClient: true
---
apiVersion: agent-install.openshift.io/v1beta1
kind: InfraEnv
metadata:
    name: site-sno-du-2
    namespace: site-sno-du-2
spec:
    additionalNTPSources:
        - NTP.server1
        - 10.16.231.22
    agentLabelSelector:
        matchLabels:
            cluster-name: site-sno-du-2
    clusterRef:
        name: site-sno-du-2
        namespace: site-sno-du-2
    ignitionConfigOverride: igen
    nmStateConfigLabelSelector:
        matchLabels:
            nmstate-label: site-sno-du-2
    pullSecretRef:
        name: pullSecretName
    sshAuthorizedKey: 'ssh-rsa '
---
apiVersion: metal3.io/v1alpha1
kind: BareMetalHost
metadata:
    annotations:
        bmac.agent-install.openshift.io/hostname: node1
        inspect.metal3.io: disabled
    labels:
        infraenvs.agent-install.openshift.io: site-sno-du-2
    name: site-sno-du-2
    namespace: site-sno-du-2
spec:
    automatedCleaningMode: disabled
    bmc:
        address: redfish-virtualmedia+https://10.16.231.87/redfish/v1/Systems/System.Embedded.1
        credentialsName: bmcSecret-du-sno2
        disableCertificateVerification: true
    bootMACAddress: "00:00:00:01:20:30"
    bootMode: UEFI
    online: true
    rootDeviceHints:
        deviceName: /dev/sdb
        model: baseModel
        vendor: sata
---
apiVersion: v1
data:
    03-sctp-machine-config.yaml: |
        apiVersion: machineconfiguration.openshift.io/v1
        kind: MachineConfig
        metadata:
          labels:
            machineconfiguration.openshift.io/role: master
          name: load-sctp-module
        spec:
          config:
            ignition:
              version: 2.2.0
            storage:
              files:
                - contents:
                    source: data:,
                    verification: {}
                  filesystem: root
                  mode: 420
                  path: /etc/modprobe.d/sctp-blacklist.conf
                - contents:
                    source: data:text/plain;charset=utf-8,sctp
                  filesystem: root
                  mode: 420
                  path: /etc/modules-load.d/sctp-load.conf
kind: ConfigMap
metadata:
    name: site-sno-du-2
    namespace: site-sno-du-2
```
# Install and execute
-  We assume kustomize and golang are installed
-  To update Policy Generator vendors and external dependencies we need to execute the following commands at the top level cnf-features-deploy directory:

    - $ cd cnf-features-deploy/
    - $ go mod tidy && go mod vendor

    - $ cd ztp/ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator/
    - $ go build -mod=vendor -o PolicyGenerator
  

- The [kustomization.yaml](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/ztp-policy-generator/kustomization.yaml) has a reference to the policyGenerator.yaml as below

```
apiVersion: policyGenerator/v1
kind: PolicyGenerator
metadata:
  name: acm-policy
  namespace: acm-policy-generator
# The arguments should be given and defined as below with same order --tempPath= --sourcePath= --outPath= --stdout
argsOneLiner: ./testPolicyGenTempExample ../source-crs ./out true
---
apiVersion: policyGenerator/v1
kind: PolicyGenerator
metadata:
  name: clusters-config
  namespace: cluster-config-generator
# The arguments should be given and defined as below with same order --siteConfigPath= --sourcePath= --outPath= --stdout
argsOneLiner: ./testSiteConfig ../source-crs ./out true
```
- The PolicyGenerator parameters are defined as below:

    - TempPath: path to the policyGenTemp files
    - sourcePath: path to the source policies
    - outPath: path to save the generated ACM policies 
    - stdout: if true will print the generated policies to console

- Test PolicyGen & siteConfig by executing the below commands:

    - $ cd cnf-features-deploy/ztp/ztp-policy-generator/
    - $ XDG_CONFIG_HOME=./ kustomize build --enable-alpha-plugins

    You should have out directory created with the expected output as below
```
├── common
│   ├── common-log-sub-policy.yaml
│   ├── common-master-mc-mount-ns-policy.yaml
│   ├── common-pao-sub-policy.yaml
│   ├── common-policies-placementbinding.yaml
│   ├── common-policies-placementrule.yaml
│   ├── common-ptp-sub-policy.yaml
│   ├── common-sriov-sub-policy.yaml
│   └── common-worker-mc-mount-ns-policy.yaml
├── customResource
│   ├── site-plan-sno-du-1
│   │   └── sno-du-1.yaml
│   └── site-plan-sno-du-2
│       └── sno-du-2.yaml
├── groups
│   ├── group-du
│   │   ├── group-du-mcp-worker-du-policy.yaml
│   │   ├── group-du-policies-placementbinding.yaml
│   │   └── group-du-policies-placementrule.yaml
│   └── group-sno-du
│       ├── group-du-sno-policies-placementbinding.yaml
│       ├── group-du-sno-policies-placementrule.yaml
│       ├── group-sno-du-console-policy.yaml
│       ├── group-sno-du-log-forwarder-policy.yaml
│       ├── group-sno-du-log-policy.yaml
│       ├── group-sno-du-mc-chronyd-policy.yaml
│       ├── group-sno-du-mc-sctp-policy.yaml
│       ├── group-sno-du-ptp-config-policy.yaml
│       └── group-sno-du-sriov-operconfig-policy.yaml
└── sites
    └── site-du-sno-1
        ├── site-du-sno-1-perfprofile-policy.yaml
        ├── site-du-sno-1-policies-placementbinding.yaml
        ├── site-du-sno-1-policies-placementrule.yaml
        ├── site-du-sno-1-sriov-nnp-fh-policy.yaml
        ├── site-du-sno-1-sriov-nnp-mh-policy.yaml
        ├── site-du-sno-1-sriov-nw-fh-policy.yaml
        └── site-du-sno-1-sriov-nw-mh-policy.yaml
```
