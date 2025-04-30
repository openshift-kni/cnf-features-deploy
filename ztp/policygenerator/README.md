## PolicyGen (Policy Generator)
The PolicyGen library is used to facilitate creating ACM policies based on a set of provided source CRs (custom resources) and [PolicyGenTemplate](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/ran-crd/policy-gen-template-crd.yaml) CR which describe how to customize those source CRs.
The full list of the CRs that ztp RAN solution provide to deploy ACM policies are in the [telco-reference/telco-ran](https://github.com/openshift-kni/telco-reference/tree/main/telco-ran/configuration/source-crs) repository. PolicyGenTemplate constructs the ACM policies by offering the following customization mechanisms:
  1. Overlay: The given CRs that will be constructed into ACM policy may have some or all of their contents replaced by values specified in the PolicyGenTemplate.
  1. Grouping: Policies defined in the PolicyGenTemplate will be created under the same namespace and share the same PlacmentRules and PlacementBinding.

By default, the policies created have `remediationAction: inform`, so that other tooling(e.g.[Topology Aware Lifecycle Operator](https://github.com/openshift-kni/cluster-group-upgrades-operator#readme)) or direct user interaction can be used to opt-in to when these policies apply to individual clusters. This can be overridden by adding `remediationAction: enforce` to the PolicyGenTemplate spec.

### Policy waves
To use the Topology Aware Lifecycle Operator roll out the policies, ZTP deploy waves are used to order how policies are applied to the spoke cluster.  All policies created by PolicyGen have a ztp deploy wave by default. The ztp deploy wave of each policy is set by using the `ran.openshift.io/ztp-deploy-wave` annotation which is based on the same wave annotation from each [source CR](https://github.com/openshift-kni/telco-reference/tree/main/telco-ran/configuration/source-crs) included in the policy. The policies have lower values should be applied first. All CRs have the same wave should be applied in the same policy. For the CRs with different waves, which means they have dependency between each other, so they are supposed to be applied in the separate policies. It's also possible to override the default source CR wave via the PolicyGenTemplate so that the CR can be included the same policy and the wave overrides should be reflected in the policy level.

### Examples
- Example 1: Consider the PolicyGenTemplate below to create ACM policies for both [DisableSnoNetworkDiag.yaml](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/source-crs/DisableSnoNetworkDiag.yaml) and [ClusterLogForwarding.yaml](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/source-crs/ClusterLogForwarding.yaml).
```
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
    - fileName: DisableSnoNetworkDiag.yaml
      policyName: "network-policy"
    - fileName: ClusterLogForwarding.yaml
      policyName: "log-forwarding-policy"
      spec:
        outputs:
        - type: "kafka"
          name: kafka-open
          kafka:
            # Example URL only
            url: tcp://192.168.1.2
        filters:
        - name: test-labels
          type: openshiftLabels
          openshiftLabels:
            label1: test1
            label2: test2
            label3: test3
            label4: test4
        pipelines:
        - name: all-to-default
          inputRefs:
          - audit
          - infrastructure
          filterRefs:
          - test-labels
          outputRefs:
          - kafka-open
```

The generated policies will be:

```
apiVersion: policy.open-cluster-management.io/v1
kind: Policy
metadata:
  annotations:
    policy.open-cluster-management.io/categories: CM Configuration Management
    policy.open-cluster-management.io/controls: CM-2 Baseline Configuration
    policy.open-cluster-management.io/standards: NIST SP 800-53
    ran.openshift.io/ztp-deploy-wave: "10"
  name: group-du-sno-network-policy
  namespace: group-du-sno-policies
spec:
  disabled: false
  policy-templates:
  - objectDefinition:
      apiVersion: policy.open-cluster-management.io/v1
      kind: ConfigurationPolicy
      metadata:
        name: group-du-sno-network-policy-config
      spec:
        evaluationInterval:
          compliant: 10m
          noncompliant: 10s
        namespaceselector:
          exclude:
          - kube-*
          include:
          - '*'
        object-templates:
        - complianceType: musthave
          objectDefinition:
            apiVersion: operator.openshift.io/v1
            kind: Network
            metadata:
              name: cluster
            spec:
              disableNetworkDiagnostics: true
        remediationAction: inform
        severity: low
  remediationAction: inform
---
apiVersion: policy.open-cluster-management.io/v1
kind: Policy
metadata:
  annotations:
    policy.open-cluster-management.io/categories: CM Configuration Management
    policy.open-cluster-management.io/controls: CM-2 Baseline Configuration
    policy.open-cluster-management.io/standards: NIST SP 800-53
    ran.openshift.io/ztp-deploy-wave: "10"
  name: group-du-sno-log-forwarding-policy
  namespace: group-du-sno-policies
spec:
  disabled: false
  policy-templates:
  - objectDefinition:
      apiVersion: policy.open-cluster-management.io/v1
      kind: ConfigurationPolicy
      metadata:
        name: group-du-sno-log-forwarding-policy-config
      spec:
        namespaceselector:
          exclude:
          - kube-*
          include:
          - '*'
        object-templates:
        - complianceType: musthave
          objectDefinition:
            apiVersion: observability.openshift.io/v1
            kind: ClusterLogForwarder
            metadata:
              name: instance
              namespace: openshift-logging
            spec:
                outputs:
                - type: "kafka"
                name: kafka-open
                kafka:
                    # Example URL only
                    url: tcp://192.168.1.2
                filters:
                - name: test-labels
                type: openshiftLabels
                openshiftLabels:
                    label1: test1
                    label2: test2
                    label3: test3
                    label4: test4
                pipelines:
                - name: all-to-default
                inputRefs:
                - audit
                - infrastructure
                filterRefs:
                - test-labels
                outputRefs:
                - kafka-open
        remediationAction: inform
        severity: low
  remediationAction: inform
```

The placement binding and rules of the generated policies will be:

```
apiVersion: policy.open-cluster-management.io/v1
kind: PlacementBinding
metadata:
  name: group-du-sno-placementbinding
  namespace: group-du-sno-policies
placementRef:
  apiGroup: apps.open-cluster-management.io
  kind: PlacementRule
  name: group-du-sno-placementrules
subjects:
- apiGroup: policy.open-cluster-management.io
  kind: Policy
  name: group-du-sno-console-policy
- apiGroup: policy.open-cluster-management.io
  kind: Policy
  name: group-du-sno-log-forwarding-policy
---
apiVersion: apps.open-cluster-management.io/v1
kind: PlacementRule
metadata:
  name: group-du-sno-placementrules
  namespace: group-du-sno-policies
spec:
  clusterSelector:
    matchExpressions:
    - key: group-du-sno
      operator: In
      values:
      - ""
```
## Build and execute
- Requirement
  - golang is installed

- Run the following command to build the policygenerator binary:
```
    $ make build
```

- Run the following command to execute the unit tests:
```
    $ make test
```

- Run the following command to execute policygenerator binary with a PolicyGenTemplate example
```
    $ ./policygenerator  -sourcePath source-crs ../ran-crd/policy-gen-template-ex.yaml
```  

- Run the following command to see the command's help text:
```
./policygenerator  --help
Usage of ./policygenerator:
  -outPath string
    	Directory to write the genrated policies (default "__unset_value__")
  -pgtPath string
    	Directory where policyGenTemp files exist (default "__unset_value__")
  -sourcePath string
    	Directory where source-crs files exist (default "source-crs")
  -wrapInPolicy
    	Wrap the CRs in acm Policy (default true)
```

- For using policygenerator library as kustomize plugin, see the [policy-generator-kustomize-plugin](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/policygenerator-kustomize-plugin/README.md). 
