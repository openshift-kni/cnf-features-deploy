package pkg

import "time"

const (
	// NamespaceTesting contains the name of the testing namespace

	ETHTOOL_HARDWARE_RECEIVE_CAP   = "hardware-receive"
	ETHTOOL_HARDWARE_TRANSMIT_CAP  = "hardware-transmit"
	ETHTOOL_HARDWARE_RAW_CLOCK_CAP = "hardware-raw-clock"
	ETHTOOL_RX_HARDWARE_FLAG       = "(SOF_TIMESTAMPING_RX_HARDWARE)"
	ETHTOOL_TX_HARDWARE_FLAG       = "(SOF_TIMESTAMPING_TX_HARDWARE)"
	ETHTOOL_RAW_HARDWARE_FLAG      = "(SOF_TIMESTAMPING_RAW_HARDWARE)"
	PtpLinuxDaemonNamespace        = "openshift-ptp"
	PtpLinuxDaemonPodsLabel        = "app=linuxptp-daemon"
	PtpOperatorDeploymentName      = "ptp-operator"
	PtPOperatorPodsLabel           = "name=ptp-operator"
	PtpDaemonsetName               = "linuxptp-daemon"

	PtpResourcesGroupVersionPrefix  = "ptp.openshift.io/v"
	PtpResourcesNameOperatorConfigs = "ptpoperatorconfigs"
	NodePtpDeviceAPIPath            = "/apis/ptp.openshift.io/v1/namespaces/openshift-ptp/nodeptpdevices/"
	ConfigPtpAPIPath                = "/apis/ptp.openshift.io/v1/namespaces/openshift-ptp/ptpconfigs"
	PtpContainerName                = "linuxptp-daemon-container"
	EventProxyContainerName         = "cloud-event-proxy"

	// policy name
	PtpGrandMasterPolicyName = "test-grandmaster"
	PtpBcMaster1PolicyName   = "test-bc-master1"
	PtpSlave1PolicyName      = "test-slave1"
	PtpBcMaster2PolicyName   = "test-bc-master2"
	PtpSlave2PolicyName      = "test-slave2"
	PtpTempPolicyName        = "temp"

	// node labels
	PtpGrandmasterNodeLabel    = "ptp/test-grandmaster"
	PtpClockUnderTestNodeLabel = "ptp/clock-under-test"
	PtpSlave1NodeLabel         = "ptp/test-slave1"
	PtpSlave2NodeLabel         = "ptp/test-slave2"
	TimeoutIn3Minutes          = 3 * time.Minute
	TimeoutIn5Minutes          = 5 * time.Minute
	TimeoutIn10Minutes         = 10 * time.Minute
	Timeout10Seconds           = 10 * time.Second
	TimeoutInterval2Seconds    = 2 * time.Second

	MasterOffsetLowerBound  = -100
	MasterOffsetHigherBound = 100

	MetricsEndPoint       = "127.0.0.1:9091/metrics"
	PtpConfigOperatorName = "default"

	RebootDaemonSetNamespace     = "ptp-reboot"
	RebootDaemonSetName          = "ptp-reboot"
	RebootDaemonSetContainerName = "container-00"

	RecoveryNetworkOutageDaemonSetNamespace     = "ptp-network-outage-recovery"
	RecoveryNetworkOutageDaemonSetName          = "ptp-network-outage-recovery"
	RecoveryNetworkOutageDaemonSetContainerName = "container-00"
)

const (
	// PtpNamespace contains the name of the ptp namespace
	PtpNamespace = "openshift-ptp"
	// NodePtpDevices contains the name of the node ptp devices CRD
	NodePtpDevicesCRD = "nodeptpdevices.ptp.openshift.io"
	// PtpConfigs contains the name of the ptp configs CRD
	PtpConfigsCRD = "ptpconfigs.ptp.openshift.io"
	// PtpOperatorConfigs contains the name of the ptp operator config CRD
	PtpOperatorConfigsCRD = "ptpoperatorconfigs.ptp.openshift.io"
)
