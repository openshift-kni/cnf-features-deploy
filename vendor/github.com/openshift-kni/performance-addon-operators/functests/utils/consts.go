package utils

import (
	"os"

	corev1 "k8s.io/api/core/v1"
)

// RoleWorkerCNF contains role name of cnf worker nodes
var RoleWorkerCNF string

// PerformanceProfileName contains the name of the PerformanceProfile created for tests
var PerformanceProfileName string

func init() {
	RoleWorkerCNF = os.Getenv("ROLE_WORKER_CNF")
	if RoleWorkerCNF == "" {
		RoleWorkerCNF = "worker-cnf"
	}

	PerformanceProfileName = os.Getenv("PERF_TEST_PROFILE")
	if PerformanceProfileName == "" {
		PerformanceProfileName = "performance"
	}
}

const (
	// RoleWorker contains the worker role
	RoleWorker = "worker"
)

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
	// PerformanceOperatorNamespace contains the name of the performance operator namespace
	PerformanceOperatorNamespace = "openshift-performance-addon"
	// NamespaceMachineConfigOperator contains the namespace of the machine-config-opereator
	NamespaceMachineConfigOperator = "openshift-machine-config-operator"
	// NamespaceTesting contains the name of the testing namespace
	NamespaceTesting = "performance-addon-operators-testing"
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
	// PerfRtKernelPrebootTuningScript contains the file name of performance pre-boot tuning script that runs on rt nodes
	PerfRtKernelPrebootTuningScript = "/usr/local/bin/pre-boot-tuning.sh"
)
