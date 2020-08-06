package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/openshift-kni/performance-addon-operators/functests/utils/discovery"
)

// RoleWorkerCNF contains role name of cnf worker nodes
var RoleWorkerCNF string

// NodeSelectorLabels contains the node labels the perfomance profile should match
var NodeSelectorLabels map[string]string

// PerformanceProfileName contains the name of the PerformanceProfile created for tests
// or an existing profile when discover mode is enabled
var PerformanceProfileName string

// NodesSelector represents the label selector used to filter impacted nodes.
var NodesSelector string

func init() {
	RoleWorkerCNF = os.Getenv("ROLE_WORKER_CNF")
	if RoleWorkerCNF == "" {
		RoleWorkerCNF = "worker-cnf"
	}

	PerformanceProfileName = os.Getenv("PERF_TEST_PROFILE")
	if PerformanceProfileName == "" {
		PerformanceProfileName = "performance"
	}

	NodesSelector = os.Getenv("NODES_SELECTOR")

	NodeSelectorLabels = map[string]string{
		fmt.Sprintf("%s/%s", LabelRole, RoleWorkerCNF): "",
	}

	if discovery.Enabled() {
		profile, err := discovery.GetDiscoveryPerformanceProfile()
		if err == nil {
			PerformanceProfileName = profile.Name
		}
		NodeSelectorLabels = profile.Spec.NodeSelector
		if NodesSelector != "" {
			keyValue := strings.Split(NodesSelector, "=")
			NodeSelectorLabels[keyValue[0]] = keyValue[1]
		}
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
)

const (
	// ContainerMachineConfigDaemon contains the name of the machine-config-daemon container
	ContainerMachineConfigDaemon = "machine-config-daemon"
)
