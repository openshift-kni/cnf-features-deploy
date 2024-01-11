package policyutils_test

import (
	"context"
	"fmt"
	"testing"

	pol "github.com/openshift-kni/cnf-features-deploy/ztp/pkg/policy-utils"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	fake "k8s.io/client-go/dynamic/fake"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	yaml "sigs.k8s.io/yaml"
)

func TestGetPoliciesForNamespace(t *testing.T) {
	var err error
	var p pol.PolicyExtractor
	p, err = initTestExtractor()
	if err != nil {
		t.Error(err)
	}

	pls, err := p.GetPoliciesForNamespace("cnfdf12")
	if len(pls) != 3 {
		t.Error("wrong number of policies read")
	}
}

func TestPolicyExtract(t *testing.T) {
	var err error
	var p pol.PolicyExtractor
	p, err = initTestExtractor()
	if err != nil {
		t.Error("failed to initialize policy interface")
	}
	pl, err := p.GetPoliciesForNamespace("cnfdf12")
	if err != nil {
		t.Error("failed to get policies")
	}
	u, err := pol.GetConfigurationObjects(pl)
	if err != nil {
		t.Error("failed to extract configuration objects")
	}
	if len(u) != 16 {
		t.Error("wrong number of objects read")
	}
}

func initTestExtractor() (pol.PolicyExtractor, error) {
	var err error
	var p pol.PolicyExtractor
	p.PolicyInterface, err = getMockPoliciesResource()
	if err != nil {
		return p, fmt.Errorf("failed to initialize policy interface")
	}
	p.Ctx = context.Background()
	return p, nil
}

func getMockPoliciesResource() (func(schema.GroupVersionResource) dynamic.NamespaceableResourceInterface, error) {
	var childPolicies []policiesv1.Policy

	const configPol = `
apiVersion: policy.open-cluster-management.io/v1
kind: Policy
metadata:
  annotations:
    argocd.argoproj.io/compare-options: IgnoreExtraneous
    policy.open-cluster-management.io/categories: CM Configuration Management
    policy.open-cluster-management.io/controls: CM-2 Baseline Configuration
    policy.open-cluster-management.io/standards: NIST SP 800-53
    ran.openshift.io/ztp-deploy-wave: "100"
  creationTimestamp: "2023-10-25T09:32:19Z"
  generation: 1
  labels:
    app.kubernetes.io/instance: policies
    policy.open-cluster-management.io/cluster-name: cnfdf12
    policy.open-cluster-management.io/cluster-namespace: cnfdf12
    policy.open-cluster-management.io/root-policy: ztp-cnfdf12-policies.cnfdf12-config-policy
  name: ztp-cnfdf12-policies.cnfdf12-config-policy
  namespace: cnfdf12
spec:
  disabled: false
  policy-templates:
  - objectDefinition:
      apiVersion: policy.open-cluster-management.io/v1
      kind: ConfigurationPolicy
      metadata:
        name: cnfdf12-config-policy-config
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
            apiVersion: sriovnetwork.openshift.io/v1
            kind: SriovNetwork
            metadata:
              name: f1u-network
              namespace: openshift-sriov-network-operator
            spec:
              networkNamespace: openshift-sriov-network-operator
              resourceName: uplane
              vlan: 140
        - complianceType: musthave
          objectDefinition:
            apiVersion: sriovnetwork.openshift.io/v1
            kind: SriovNetworkNodePolicy
            metadata:
              name: sriov-nnp-uplane
              namespace: openshift-sriov-network-operator
            spec:
              deviceType: vfio-pci
              isRdma: false
              nicSelector:
                pfNames:
                - ens2f1
              nodeSelector:
                node-role.kubernetes.io/master: ""
              numVfs: 8
              priority: 10
              resourceName: uplane
        - complianceType: musthave
          objectDefinition:
            apiVersion: sriovnetwork.openshift.io/v1
            kind: SriovNetwork
            metadata:
              name: f1c-network
              namespace: openshift-sriov-network-operator
            spec:
              networkNamespace: openshift-sriov-network-operator
              resourceName: cplane
              vlan: 150
        - complianceType: musthave
          objectDefinition:
            apiVersion: sriovnetwork.openshift.io/v1
            kind: SriovNetworkNodePolicy
            metadata:
              name: sriov-nnp-cplane
              namespace: openshift-sriov-network-operator
            spec:
              deviceType: netdevice
              isRdma: true
              nicSelector:
                pfNames:
                - ens2f0
              nodeSelector:
                node-role.kubernetes.io/master: ""
              numVfs: 8
              priority: 10
              resourceName: cplane
        remediationAction: inform
        severity: low
  remediationAction: inform
status:
  compliant: NonCompliant
  details:
  - compliant: NonCompliant
    history:
    - eventName: ztp-cnfdf12-policies.cnfdf12-config-policy.17914fdcf51a3351
      lastTimestamp: "2023-10-25T09:32:24Z"
      message: NonCompliant; violation - couldn't find mapping resource with kind
        SriovNetwork, please check if you have CRD deployed; violation - couldn't
        find mapping resource with kind SriovNetworkNodePolicy, please check if you
        have CRD deployed; violation - couldn't find mapping resource with kind SriovNetwork,
        please check if you have CRD deployed; violation - couldn't find mapping resource
        with kind SriovNetworkNodePolicy, please check if you have CRD deployed
    templateMeta:
      creationTimestamp: null
      name: cnfdf12-config-policy-config

`

	const groupPol = `
apiVersion: policy.open-cluster-management.io/v1
kind: Policy
metadata:
  annotations:
    argocd.argoproj.io/compare-options: IgnoreExtraneous
    policy.open-cluster-management.io/categories: CM Configuration Management
    policy.open-cluster-management.io/controls: CM-2 Baseline Configuration
    policy.open-cluster-management.io/standards: NIST SP 800-53
    ran.openshift.io/ztp-deploy-wave: "10"
  creationTimestamp: "2023-10-25T09:32:19Z"
  generation: 1
  labels:
    app.kubernetes.io/instance: policies
    policy.open-cluster-management.io/cluster-name: cnfdf12
    policy.open-cluster-management.io/cluster-namespace: cnfdf12
    policy.open-cluster-management.io/root-policy: ztp-group-cnfdf12.group-cnfdf12-config-policy
  name: ztp-group-cnfdf12.group-cnfdf12-config-policy
  namespace: cnfdf12
spec:
  disabled: false
  policy-templates:
  - objectDefinition:
      apiVersion: policy.open-cluster-management.io/v1
      kind: ConfigurationPolicy
      metadata:
        name: group-cnfdf12-config-policy-config
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
            apiVersion: performance.openshift.io/v2
            kind: PerformanceProfile
            metadata:
              annotations:
                ran.openshift.io/reference-configuration: ran-du.redhat.com
              name: openshift-node-performance-profile
            spec:
              additionalKernelArgs:
              - rcupdate.rcu_normal_after_boot=0
              - efi=runtime
              - vfio_pci.enable_sriov=1
              - vfio_pci.disable_idle_d3=1
              - module_blacklist=irdma
              cpu:
                isolated: 4-47
                reserved: 0-3
              hugepages:
                defaultHugepagesSize: 1G
                pages:
                - count: 32
                  size: 1G
              machineConfigPoolSelector:
                pools.operator.machineconfiguration.openshift.io/master: ""
              nodeSelector:
                node-role.kubernetes.io/master: ""
              numa:
                topologyPolicy: restricted
              realTimeKernel:
                enabled: true
              workloadHints:
                highPowerConsumption: false
                perPodPowerManagement: false
                realTime: true
        - complianceType: musthave
          objectDefinition:
            apiVersion: tuned.openshift.io/v1
            kind: Tuned
            metadata:
              name: performance-patch
              namespace: openshift-cluster-node-tuning-operator
            spec:
              profile:
              - data: |
                  [main]
                  summary=Configuration changes profile inherited from performance created tuned
                  include=openshift-node-performance-openshift-node-performance-profile
                  [bootloader]
                  cmdline_crash=nohz_full=4-47
                  [sysctl]
                  kernel.timer_migration=1
                  [scheduler]
                  group.ice-ptp=0:f:10:*:ice-ptp.*
                  [service]
                  service.stalld=start,enable
                  service.chronyd=stop,disable
                name: performance-patch
              recommend:
              - machineConfigLabels:
                  machineconfiguration.openshift.io/role: master
                priority: 19
                profile: performance-patch
        - complianceType: musthave
          objectDefinition:
            apiVersion: ptp.openshift.io/v1
            kind: PtpConfig
            metadata:
              name: du-ptp-slave
              namespace: openshift-ptp
            spec:
              profile:
              - interface: ens1f1
                name: slave
                phc2sysOpts: -a -r -n 24
                ptp4lConf: |
                  [global]
                  #
                  # Default Data Set
                  #
                  twoStepFlag 1
                  slaveOnly 1
                  priority1 128
                  priority2 128
                  domainNumber 24
                  #utc_offset 37
                  clockClass 255
                  clockAccuracy 0xFE
                  offsetScaledLogVariance 0xFFFF
                  free_running 0
                  freq_est_interval 1
                  dscp_event 0
                  dscp_general 0
                  dataset_comparison G.8275.x
                  G.8275.defaultDS.localPriority 128
                  #
                  # Port Data Set
                  #
                  logAnnounceInterval -3
                  logSyncInterval -4
                  logMinDelayReqInterval -4
                  logMinPdelayReqInterval -4
                  announceReceiptTimeout 3
                  syncReceiptTimeout 0
                  delayAsymmetry 0
                  fault_reset_interval -4
                  neighborPropDelayThresh 20000000
                  masterOnly 0
                  G.8275.portDS.localPriority 128
                  #
                  # Run time options
                  #
                  assume_two_step 0
                  logging_level 6
                  path_trace_enabled 0
                  follow_up_info 0
                  hybrid_e2e 0
                  inhibit_multicast_service 0
                  net_sync_monitor 0
                  tc_spanning_tree 0
                  tx_timestamp_timeout 50
                  unicast_listen 0
                  unicast_master_table 0
                  unicast_req_duration 3600
                  use_syslog 1
                  verbose 0
                  summary_interval 0
                  kernel_leap 1
                  check_fup_sync 0
                  clock_class_threshold 7
                  #
                  # Servo Options
                  #
                  pi_proportional_const 0.0
                  pi_integral_const 0.0
                  pi_proportional_scale 0.0
                  pi_proportional_exponent -0.3
                  pi_proportional_norm_max 0.7
                  pi_integral_scale 0.0
                  pi_integral_exponent 0.4
                  pi_integral_norm_max 0.3
                  step_threshold 2.0
                  first_step_threshold 0.00002
                  max_frequency 900000000
                  clock_servo pi
                  sanity_freq_limit 200000000
                  ntpshm_segment 0
                  #
                  # Transport options
                  #
                  transportSpecific 0x0
                  ptp_dst_mac 01:1B:19:00:00:00
                  p2p_dst_mac 01:80:C2:00:00:0E
                  udp_ttl 1
                  udp6_scope 0x0E
                  uds_address /var/run/ptp4l
                  #
                  # Default interface options
                  #
                  clock_type OC
                  network_transport L2
                  delay_mechanism E2E
                  time_stamping hardware
                  tsproc_mode filter
                  delay_filter moving_median
                  delay_filter_length 10
                  egressLatency 0
                  ingressLatency 0
                  boundary_clock_jbod 0
                  #
                  # Clock description
                  #
                  productDescription ;;
                  revisionData ;;
                  manufacturerIdentity 00:00:00
                  userDescription ;
                  timeSource 0xA0
                ptp4lOpts: -2 -s --summary_interval 0
                ptpSchedulingPolicy: SCHED_FIFO
                ptpSchedulingPriority: 10
                ptpSettings:
                  logReduce: "true"
              recommend:
              - match:
                - nodeLabel: node-role.kubernetes.io/master
                priority: 4
                profile: slave
        remediationAction: inform
        severity: low
  remediationAction: inform
status:
  compliant: NonCompliant
  details:
  - compliant: NonCompliant
    history:
    - eventName: ztp-group-cnfdf12.group-cnfdf12-config-policy.17914fdcf6acdd93
      lastTimestamp: "2023-10-25T09:32:24Z"
      message: 'NonCompliant; violation - performanceprofiles not found: [openshift-node-performance-profile]
        missing; violation - tuneds not found: [performance-patch] in namespace openshift-cluster-node-tuning-operator
        missing; violation - couldn''t find mapping resource with kind PtpConfig,
        please check if you have CRD deployed'
    templateMeta:
      creationTimestamp: null
      name: group-cnfdf12-config-policy-config

`

	const commonPol = `
apiVersion: policy.open-cluster-management.io/v1
kind: Policy
metadata:
  annotations:
    argocd.argoproj.io/compare-options: IgnoreExtraneous
    policy.open-cluster-management.io/categories: CM Configuration Management
    policy.open-cluster-management.io/controls: CM-2 Baseline Configuration
    policy.open-cluster-management.io/standards: NIST SP 800-53
    ran.openshift.io/ztp-deploy-wave: "2"
  labels:
    app.kubernetes.io/instance: policies
    policy.open-cluster-management.io/cluster-name: cnfdf12
    policy.open-cluster-management.io/cluster-namespace: cnfdf12
    policy.open-cluster-management.io/root-policy: ztp-common-cnfdf12.common-cnfdf12-subscriptions-policy
  name: ztp-common-cnfdf12.common-cnfdf12-subscriptions-policy
  namespace: cnfdf12
  resourceVersion: "48291735"
  uid: a2698dcf-4b7d-4eb9-abde-b74b0d587600
spec:
  disabled: false
  policy-templates:
  - objectDefinition:
      apiVersion: policy.open-cluster-management.io/v1
      kind: ConfigurationPolicy
      metadata:
        name: common-cnfdf12-subscriptions-policy-config
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
            apiVersion: operators.coreos.com/v1alpha1
            kind: Subscription
            metadata:
              name: sriov-network-operator-subscription
              namespace: openshift-sriov-network-operator
            spec:
              channel: stable
              installPlanApproval: Manual
              name: sriov-network-operator
              source: redhat-operators-413
              sourceNamespace: openshift-marketplace
            status:
              state: AtLatestKnown
        - complianceType: musthave
          objectDefinition:
            apiVersion: v1
            kind: Namespace
            metadata:
              annotations:
                workload.openshift.io/allowed: management
              name: openshift-sriov-network-operator
        - complianceType: musthave
          objectDefinition:
            apiVersion: operators.coreos.com/v1
            kind: OperatorGroup
            metadata:
              name: sriov-network-operators
              namespace: openshift-sriov-network-operator
            spec:
              targetNamespaces:
              - openshift-sriov-network-operator
        - complianceType: musthave
          objectDefinition:
            apiVersion: v1
            kind: Namespace
            metadata:
              annotations:
                workload.openshift.io/allowed: management
              name: openshift-local-storage
        - complianceType: musthave
          objectDefinition:
            apiVersion: operators.coreos.com/v1
            kind: OperatorGroup
            metadata:
              name: openshift-local-storage
              namespace: openshift-local-storage
            spec:
              targetNamespaces:
              - openshift-local-storage
        - complianceType: musthave
          objectDefinition:
            apiVersion: operators.coreos.com/v1alpha1
            kind: Subscription
            metadata:
              name: local-storage-operator
              namespace: openshift-local-storage
            spec:
              channel: stable
              installPlanApproval: Manual
              name: local-storage-operator
              source: redhat-operators-413
              sourceNamespace: openshift-marketplace
            status:
              state: AtLatestKnown
        - complianceType: musthave
          objectDefinition:
            apiVersion: operators.coreos.com/v1alpha1
            kind: Subscription
            metadata:
              name: ptp-operator-subscription
              namespace: openshift-ptp
            spec:
              channel: stable
              installPlanApproval: Manual
              name: ptp-operator
              source: redhat-operators-413
              sourceNamespace: openshift-marketplace
            status:
              state: AtLatestKnown
        - complianceType: musthave
          objectDefinition:
            apiVersion: v1
            kind: Namespace
            metadata:
              annotations:
                workload.openshift.io/allowed: management
              labels:
                openshift.io/cluster-monitoring: "true"
              name: openshift-ptp
        - complianceType: musthave
          objectDefinition:
            apiVersion: operators.coreos.com/v1
            kind: OperatorGroup
            metadata:
              name: ptp-operators
              namespace: openshift-ptp
            spec:
              targetNamespaces:
              - openshift-ptp
        remediationAction: inform
        severity: low
  remediationAction: inform
status:
  compliant: NonCompliant
  details:
  - compliant: NonCompliant
    history:
    - eventName: ztp-common-cnfdf12.common-cnfdf12-subscriptions-policy.17914fdcf6d77a29
      lastTimestamp: "2023-10-25T09:32:24Z"
      message: 'NonCompliant; violation - subscriptions not found: [sriov-network-operator-subscription]
        in namespace openshift-sriov-network-operator missing; violation - namespaces
        not found: [openshift-sriov-network-operator] missing; violation - operatorgroups
        not found: [sriov-network-operators] in namespace openshift-sriov-network-operator
        missing; violation - namespaces not found: [openshift-local-storage] missing;
        violation - operatorgroups not found: [openshift-local-storage] in namespace
        openshift-local-storage missing; violation - subscriptions not found: [local-storage-operator]
        in namespace openshift-local-storage missing; violation - subscriptions not
        found: [ptp-operator-subscription] in namespace openshift-ptp missing; violation
        - namespaces not found: [openshift-ptp] missing; violation - operatorgroups
        not found: [ptp-operators] in namespace openshift-ptp missing'
    templateMeta:
      creationTimestamp: null
      name: common-cnfdf12-subscriptions-policy-config

`
	for _, item := range []string{configPol, groupPol, commonPol} {
		var pol policiesv1.Policy
		err := yaml.Unmarshal([]byte(item), &pol)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal ")
		}
		childPolicies = append(childPolicies, *pol.DeepCopy())
	}
	fakeclient := fake.NewSimpleDynamicClient(runtime.NewScheme(), childPolicies[0].DeepCopy(), childPolicies[1].DeepCopy(), childPolicies[2].DeepCopy())
	rv := fakeclient.Resource
	return rv, nil
}
