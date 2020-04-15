package components

const (
	// AssetsDir defines the directory with assets under the operator image
	AssetsDir = "/assets"
)

const (
	// ComponentNamePrefix defines the worker role for performance sensitive workflows
	// TODO: change it back to longer name once https://bugzilla.redhat.com/show_bug.cgi?id=1787907 fixed
	// ComponentNamePrefix = "worker-performance"
	ComponentNamePrefix = "performance"
	// MachineConfigRoleLabelKey is the label key to use as label and in MachineConfigSelector of MCP which targets the performance profile
	MachineConfigRoleLabelKey = "machineconfiguration.openshift.io/role"
)

const (
	// NamespaceNodeTuningOperator defines the tuned profiles namespace
	NamespaceNodeTuningOperator = "openshift-cluster-node-tuning-operator"
	// ProfileNamePerformance defines the performance tuned profile name
	ProfileNamePerformance = "openshift-node-performance"
)

const (
	// FeatureGateLatencySensetiveName defines the latency sensetive feature gate name
	// TOOD: uncomment once https://bugzilla.redhat.com/show_bug.cgi?id=1788061 fixed
	// FeatureGateLatencySensetiveName = "latency-sensitive"
	FeatureGateLatencySensetiveName = "cluster"
)
