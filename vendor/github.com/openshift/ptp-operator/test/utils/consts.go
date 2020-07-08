package utils

const (
	// NamespaceTesting contains the name of the testing namespace
	NamespaceTesting = "ptp-testing"

	ETHTOOL_HARDWARE_RECEIVE_CAP    = "hardware-receive"
	ETHTOOL_HARDWARE_TRANSMIT_CAP   = "hardware-transmit"
	ETHTOOL_HARDWARE_RAW_CLOCK_CAP  = "hardware-raw-clock"
	ETHTOOL_RX_HARDWARE_FLAG        = "(SOF_TIMESTAMPING_RX_HARDWARE)"
	ETHTOOL_TX_HARDWARE_FLAG        = "(SOF_TIMESTAMPING_TX_HARDWARE)"
	ETHTOOL_RAW_HARDWARE_FLAG       = "(SOF_TIMESTAMPING_RAW_HARDWARE)"
	PtpLinuxDaemonNamespace         = "openshift-ptp"
	PtpOperatorDeploymentName       = "ptp-operator"
	PtpDaemonsetName                = "linuxptp-daemon"
	PtpSlaveNodeLabel               = "ptp/test-slave"
	PtpGrandmasterNodeLabel         = "ptp/test-grandmaster"
	PtpResourcesGroupVersionPrefix  = "ptp.openshift.io/v"
	PtpResourcesNameOperatorConfigs = "ptpoperatorconfigs"
	NodePtpDeviceAPIPath            = "/apis/ptp.openshift.io/v1/namespaces/openshift-ptp/nodeptpdevices/"
	ConfigPtpAPIPath                = "/apis/ptp.openshift.io/v1/namespaces/openshift-ptp/ptpconfigs"
	PtpContainerName                = "linuxptp-daemon-container"
)

var (
	PtpGrandMasterPolicyName = "test-grandmaster"
	PtpSlavePolicyName       = "test-slave"
)
