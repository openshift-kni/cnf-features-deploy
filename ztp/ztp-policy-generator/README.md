# Policy Generator (PolicyGen)

The policy generator (PolicyGen) is a kustomize plugin used to facilitate creating ACM policies from a predefined Custom Resources (CRs). There are 3 main items (Policy Categorization, Source CR policy and PolicyGenTemplate) that PolicyGen rely on to generate the ACM policies and its placement binding and placement rule.

1 - Policy Categorization:

 Any generated policy needs to be placed under one of the following categories; 

   - Common: a policy exist under the common category will be applied to all clusters.

   - Groups: a policy exist under the groups category will be applied to group of clusters. Every group of clusters could have their own policies exist under the groups category. Ex; Groups/group1 will has its own policies that get applied to the clusters belong to group1.

   - Sites: a policy exist under the sites category will be applied to a specific cluster. Any cluster could has its own policies exist under the sites category. Ex; Sites/cluster1 will has its own policies get applied to cluster1

2 - Source CR policy:

The source CR that will be used to generate the ACM policy needs to be defined with consideration of possible overlay to its metadata or spec/data. For example; a common-namespace-policy contain a Namespace definition that will exist in all managed clusters. This namespace will be placed under the common category and there will be no changes for its spec or data across all clusters. The source CR for this namespace will be as below

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
                   - complianceType: musthave
                     objectDefinition:
                       apiVersion: v1
                       kind: Namespace
                       metadata:
                           labels:
                               openshift.io/run-level: "1"
                           name: openshift-sriov-network-operator
```

Another example; a SriovNetworkNodePolicy definition that will be exist in different clusters with different spec for each cluster. The source CR for the SriovNetworkNodePolicy will be as below.

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

PolicyGenTemplate is a Custom Resource Definition (CRD) that tells PolicyGen where to locate the generated policies and which Spec items need to be defined. Check the [PolicyGenTemplate](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/ztp-policy-generator/policyGenTemplates/policyGenTemplate.yaml) for more info. Let's consider the [group-du-ranGen.yaml](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/ztp-policy-generator/ranPolicyGenTemplateExamples/group-du-ranGen.yaml) example to explain more

```
apiVersion: policyGenerator/v1
kind: PolicyGenTemplate
metadata:
  name: "group-du-policies"
  namespace: "policy-template"
  labels:
    common: false
    groupName: "group-du"
    siteName: "N/A"
    mcp: "worker-du"
sourceFiles:
  - fileName: MachineConfigPool
    policyName: "mcp-du-policy"
    name: "worker-du"
  - fileName: SriovOperatorConfig
    policyName: "sriov-operconfig-policy"
  - fileName: MachineConfigSctp
    policyName: "mc-sctp-policy"
  - fileName: MachineConfigContainerMountNS
    policyName: "mc-mount-ns-policy"
  - fileName: MachineConfigDisableChronyd
    policyName: "mc-chronyd-policy"
  - fileName: PtpConfigSlave
    policyName: "ptp-config-policy"
    name: "du-ptp-slave"
    spec:
      profile:
      - name: "slave"
        interface: "ens5f0"
        ptp4lOpts: "-2 -s --summary_interval -4"
        phc2sysOpts: "-a -r -n 24"
```

The group-du-ranGen.yaml defines group of policies under a group named group-du. It defines a MachineConfigPool worker-du that will be used as the node selector for any other policy defined under the sourceFiles. For every source file exist under sourceFiles an ACM policy will be generated. And a single placement binding and placement rule will be generated to apply the cluster selection rule for group-du policies. Let's consider the source file [PtpConfigSlave](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/ztp-policy-generator/sourcePolicies/PtpConfigSlave.yaml) as example; the PtpConfigSlave has a definition of a PtpConfig CR. The generated policy for the PtpConfigSlave example will be named as group-du-ptp-config-policy. The PtpConfig CR defined in the generated group-du-ptp-config-policy will be named as du-ptp-slave. The spec defined in the PtpConfigSlave will be placed under du-ptp-slave along with the other spec items defined under the source file. The group-du-ptp-config-policy will be as below

```
apiVersion: policy.open-cluster-management.io/v1
kind: Policy
metadata:
    name: group-du-ptp-config-policy
    namespace: groups-sub
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
                name: group-du-ptp-config-policy-config
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
                        apiVersion: ptp.openshift.io/v1
                        kind: PtpConfig
                        metadata:
                            name: slave
                            namespace: openshift-ptp
                        spec:
                            recommend:
                                - match:
                                - nodeLabel: node-role.kubernetes.io/worker-du
                                  priority: 4
                                  profile: slave
                            profile:
                                - interface: ens5f0
                                  name: slave
                                  phc2sysOpts: -a -r -n 24
                                  ptp4lConf: |
                                    [global]
                                    #
                                    # Default Data Set
                                    #
                                    twoStepFlag 1
                                    slaveOnly 0
                                    priority1 128
                                    priority2 128
                                    domainNumber 24
                                    .....
```

# Site Config generator

Site config generator uses the SiteConfig CR to generate the required CRs to create an Openshift cluster using ACM operator OR Assisted-installer operator. Given the examples under siteConfigExamples/ the generated CRs will be place under out/customResource directory after executing the kustomize command with the policy generator plugin. Check policyGenerator.yaml as an example.

# Update cached dependencies
- Policy Generator vendors any external dependencies. To update any external dependencies added or changed during development, we need to execute the following commands at the top level cnf-features-deploy directory:
  
  - $ go mod tidy && go mod vendor
  
# Install and execute
-  We assume kustomize and golang are installed
-  Build the plugin

    - $ cd ztp/ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator/
    - $ go build -mod=vendor -o PolicyGenerator
  

- The [kustomization.yaml](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/ztp-policy-generator/kustomization.yaml) has a reference to the policyGenerator.yaml . The PolicyGenerator definition as below

```
apiVersion: policyGenerator/v1
kind: PolicyGenerator
metadata:
  name: acm-policy
  namespace: acm-policy-generator
# The arguments should be given and defined as below with same order --policyGenTempPath= --sourcePath= --outPath= --stdout --customResources --siteconfig
argsOneLiner: ./ranPolicyGenTempExamples ./sourcePolicies ./out true false
```
- The PolicyGenerator parameters are defined as below:

    - policyGenTempPath: path to the policyGenTemp files
    - sourcePath: path to the source policies
    - outPath: path to save the generated ACM policies 
    - stdout: if true will print the generated policies to console
    - customResourceOnly: if true will generate the CRs from the source Policies files only without ACM policies.
    - siteconfig: if true will generate the CRs for the given siteConfig CRs

- Test PolicyGen by executing the below commands:

    - $ cd cnf-features-deploy/ztp/ztp-policy-generator/
    - $ XDG_CONFIG_HOME=./ kustomize build --enable-alpha-plugins

    You should have out directory created with the expected policies as below
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
As you can see the common policies are flat because they will be applied to all clusters. However, the groups and sites have sub directories for each group and site as they will be applied to different clusters.
