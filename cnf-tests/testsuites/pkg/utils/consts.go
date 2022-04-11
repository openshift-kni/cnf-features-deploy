package utils

import corev1 "k8s.io/api/core/v1"

const (
	// LabelRole contains the key for the role label
	LabelRole = "node-role.kubernetes.io"
	// LabelHostname contains the key for the hostname label
	LabelHostname = "kubernetes.io/hostname"
)

const (
	// ResourceSRIOV contains the name of SRIOV resource under the node
	ResourceSRIOV = corev1.ResourceName("openshift.io/sriovnic")
)

const (
	// NamespaceMachineConfigOperator contains the namespace of the machine-config-opereator
	NamespaceMachineConfigOperator = "openshift-machine-config-operator"
	// NamespaceTesting contains the name of the testing namespace
	NamespaceTesting = "cnf-features-testing"
	// NamespaceOvn contains the namespace of OVN related resources
	NamespaceOvn = "openshift-ovn-kubernetes"
	// OperatorNamespace contains the name of the openshift operator namespace
	OperatorNamespace = "openshift-operators"
)

const (
	// FilePathKubeletConfig contains the kubelet.conf file path
	FilePathKubeletConfig = "/etc/kubernetes/kubelet.conf"
	// FilePathSRIOVDevice contains SRIOV device file path
	FilePathSRIOVDevice = "/sys/bus/pci/drivers/vfio-pci"
	// FilePathKubePodsSlice contains cgroup kubepods.slice file path
	FilePathKubePodsSlice = "/sys/fs/cgroup/cpuset/kubepods.slice"
	// FilePathSysCPU contains system CPU device file path
	FilePathSysCPU = "/sys/devices/system/cpu"
)

const (
	// FeatureGateTopologyManager contains topology manager feature gate name
	FeatureGateTopologyManager = "TopologyManager"
)

const (
	// EnvPciSriovDevice contains the ENV variable name of SR-IOV PCI device
	EnvPciSriovDevice = "PCIDEVICE_OPENSHIFT_IO_SRIOVNIC"
)

const (
	// ContainerMachineConfigDaemon contains the name of the machine-config-daemon container
	ContainerMachineConfigDaemon = "machine-config-daemon"
)

const (
	// PerformanceCRDName contains the name of the performance profile CRD
	PerformanceCRDName = "performanceprofiles.performance.openshift.io"
)

const (
	// SriovOperatorDeploymentName contains the name of the sriov operator deployment
	SriovOperatorDeploymentName = "sriov-network-operator"

	// SriovNetworkNodePolicies contains the name of the sriov network node policies CRD
	SriovNetworkNodePolicies = "sriovnetworknodepolicies.sriovnetwork.openshift.io"
	// SriovNetworkNodeStates contains the name of the sriov network node state CRD
	SriovNetworkNodeStates = "sriovnetworknodestates.sriovnetwork.openshift.io"
	// SriovNetworks contains the name of the sriov network CRD
	SriovNetworks = "sriovnetworks.sriovnetwork.openshift.io"
	// SriovOperatorConfigs contains the name of the sriov Operator config CRD
	SriovOperatorConfigs = "sriovoperatorconfigs.sriovnetwork.openshift.io"
	// SriovSupportedNicsCM contains the name of the sriov supported nics ConfigMap
	SriovSupportedNicsCM = "supported-nic-ids"
)

const (
	// PtpNamespace contains the name of the ptp namespace
	PtpNamespace = "openshift-ptp"
	// PtpOperatorDeploymentName contains the name of the ptp operator deployment
	PtpOperatorDeploymentName = "ptp-operator"
	// PtpDaemonsetName contains the name of the linuxptp daemonset
	PtpDaemonsetName = "linuxptp-daemon"

	// NodePtpDevices contains the name of the node ptp devices CRD
	NodePtpDevices = "nodeptpdevices.ptp.openshift.io"
	// PtpConfigs contains the name of the ptp configs CRD
	PtpConfigs = "ptpconfigs.ptp.openshift.io"
	// PtpOperatorConfigs contains the name of the ptp operator config CRD
	PtpOperatorConfigs = "ptpoperatorconfigs.ptp.openshift.io"
)

const (
	// N3000DeploymentName contains the name of the n3000 deployment
	N3000DeploymentName = "n3000-controller-manager"
	// N3000DaemonsetDriverName contains the name of the n3000 driver daemonset
	N3000DaemonsetDriverName = "fpga-driver-daemonset"
	// N3000DaemonsetTelemetryName contains the name of the n3000 telemetry daemonset
	N3000DaemonsetTelemetryName = "fpgainfo-exporter"
	// N3000DaemonsetN3000DaemonName contains the name of the n3000 daemon daemonset
	N3000DaemonsetN3000DaemonName = "n3000-daemonset"
	// N3000DaemonsetDiscoveryName contains the name of the n3000 discovery daemonset
	N3000DaemonsetDiscoveryName = "accelerator-discovery"
	// N3000NodeCRDName contains the name of the n3000node policies CRD
	N3000NodeCRDName = "n3000nodes.fpga.intel.com"
	// N3000ClusterCRDName contains the name of the n3000 cluster policies CRD
	N3000ClusterCRDName = "n3000clusters.fpga.intel.com"
)

const (
	// SriovFecDeploymentName contains the name of the sriov-fec deployment
	SriovFecDeploymentName = "sriov-fec-controller-manager"
	// SriovFecDaemonsetPluginName contains the name of the sriov plugin
	SriovFecDaemonsetPluginName = "sriov-device-plugin"
	// SriovFecDaemonsetName contains the name of sriov-fec daemonset
	SriovFecDaemonsetName = "sriov-fec-daemonset"
	// SriovFecNodeConfigCRDName contains the name of the SriovFecNode Config policies CRD
	SriovFecNodeConfigCRDName = "sriovfecnodeconfigs.sriovfec.intel.com"
	// SriovFecClusterConfigCRDName contains the name of the SriovFecCluster config policies CRD
	SriovFecClusterConfigCRDName = "sriovfecclusterconfigs.sriovfec.intel.com"
)

const (
	// GatekeeperNamespace contains the name of the gatekeeper namespace
	GatekeeperNamespace = "openshift-gatekeeper-system"
	// GatekeeperAuditDeploymentName contains the name of the gatekeeper-audit deployment
	GatekeeperAuditDeploymentName = "gatekeeper-audit"
	// GatekeeperControllerDeploymentName contains the name of the gatekeeper-controller-manager deployment
	GatekeeperControllerDeploymentName = "gatekeeper-controller-manager"
	// GatekeeperOperatorDeploymentName contains the name of the gatekeeper-operator-controller-manager deployment
	GatekeeperOperatorDeploymentName = "gatekeeper-operator-controller"
	// GatekeeperTestingNamespace is the namespace for resources in this test
	GatekeeperTestingNamespace = "gatekeeper-testing"
	// GatekeeperMutationIncludedNamespace is a test namespace that includes mutation
	GatekeeperMutationIncludedNamespace = "mutation-included"
	// GatekeeperMutationExcludedNamespace is a test namespace that is excluded from mutation
	GatekeeperMutationExcludedNamespace = "mutation-excluded"
	// GatekeeperMutationEnabledNamespace is a test namespace with mutation enabled
	GatekeeperMutationEnabledNamespace = "mutation-enabled"
	// GatekeeperMutationDisabledNamespace is a test namespace with mutation disabled
	GatekeeperMutationDisabledNamespace = "mutation-disabled"
	// GatekeeperTestObjectNamespace is a test namespace used as a mutated runtime object
	GatekeeperTestObjectNamespace = "gk-test-object"
	// GatekeeperConstraintValidationNamespace is a test namespace used to test constraints
	GatekeeperConstraintValidationNamespace = "gk-constraint-validation"
)

const (
	// NfdNodeFeatureDiscoveryCRDName node feature discovery crd name
	NfdNodeFeatureDiscoveryCRDName = "nodefeaturediscoveries.nfd.openshift.io"
	// NfdNamespace node feature discovery operator namespace
	NfdNamespace = "openshift-nfd"
	// NfdOperatorDeploymentName node feature discovery operator deployment name
	NfdOperatorDeploymentName = "nfd-controller-manager"
	// NfdMasterNodeDaemonsetName node feature discovery daemonset name for master nodes
	NfdMasterNodeDaemonsetName = "nfd-master"
	// NfdWorkerNodeDaemonsetName node feature discovery daemonset name for worker nodes
	NfdWorkerNodeDaemonsetName = "nfd-worker"
	// SroSpecialResourceCRDName special resource operator crd name
	SroSpecialResourceCRDName = "specialresources.sro.openshift.io"
	// SroOperatorDeploymentName special resource operator deployment name
	SroOperatorDeploymentName = "special-resource-controller-manager"
)
