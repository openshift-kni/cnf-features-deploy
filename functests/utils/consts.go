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
	// PerformanceOperatorNamespace contains the name of the performance operator namespace
	PerformanceOperatorNamespace = "openshift-performance-addon"
	// PerformanceOperatorDeploymentName contains the name of the performance operator deployment
	PerformanceOperatorDeploymentName = "performance-operator"
	// PerformanceCRDName contains the name of the performance profile CRD
	PerformanceCRDName = "performanceprofiles.performance.openshift.io"
)

const (
	// SriovNamespace contains the name of the sriov namespace
	SriovNamespace = "openshift-sriov-network-operator"
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
