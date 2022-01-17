package policyGen

const acmPolicyTemplate = `
apiVersion: policy.open-cluster-management.io/v1
kind: Policy
metadata:
    name: policyGenTemplate.metadata.name-sourceFiles.policyName
    namespace: policyGenTemplate.metadata.namespace
    annotations:
        policy.open-cluster-management.io/categories: CM Configuration Management
        policy.open-cluster-management.io/controls: CM-2 Baseline Configuration
        policy.open-cluster-management.io/standards: NIST SP 800-53
spec:
    remediationAction: inform
    disabled: false
    policy-templates:
        - objectDefinition:
            apiVersion: policy.open-cluster-management.io/v1
            kind: ConfigurationPolicy
            metadata:
                name: policyGenTemplate.metadata.name-sourceFiles.policyName-config
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
`
