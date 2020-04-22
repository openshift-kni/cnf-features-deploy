package utils

import (
	"os"

	corev1 "k8s.io/api/core/v1"
)

// RoleWorkerRT contains the worker-rt role
var RoleWorkerRT string

func init() {
	RoleWorkerRT = os.Getenv("ROLE_WORKER_RT")
	if RoleWorkerRT == "" {
		RoleWorkerRT = "worker-rt"
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
