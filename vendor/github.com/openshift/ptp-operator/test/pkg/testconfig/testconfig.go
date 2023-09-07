package testconfig

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"context"

	ptpv1 "github.com/openshift/ptp-operator/api/v1"
	ptptestconfig "github.com/openshift/ptp-operator/test/conformance/config"
	"github.com/openshift/ptp-operator/test/pkg"
	"github.com/openshift/ptp-operator/test/pkg/clean"
	"github.com/openshift/ptp-operator/test/pkg/client"
	"github.com/openshift/ptp-operator/test/pkg/nodes"
	"github.com/openshift/ptp-operator/test/pkg/ptphelper"
	"github.com/sirupsen/logrus"
	solver "github.com/test-network-function/graphsolver-lib"
	l2lib "github.com/test-network-function/l2discovery-lib"
	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (

	// DualNICBoundaryClockString matches the Discovery clock mode in Environement
	DiscoveryString = "Discovery"
	// legacyDiscoveryString matches the legacy Discovery clock mode in Environement
	legacyDiscoveryString = "true"
	// NoneString matches empty environment variable
	NoneString = ""
	// StartString Stringer value for the start status
	StartString = "Start"
	// InitStatusString Stringer value for the init status
	InitStatusString = "init"
	// ConfiguredStatusString Stringer value for the configured status
	ConfiguredStatusString = "configured"
	// DiscoverySuccessStatusString Stringer value for the Discovery Success status
	DiscoverySuccessStatusString = "discoverySuccess"
	// DiscoveryFailureStatusString Stringer value for the Discovery failure status
	DiscoveryFailureStatusString = "discoveryFailure"
	PtpLinuxDaemonNamespace      = "openshift-ptp"
	int65                        = 65
	int5                         = 5
	// OrdinaryClockString matches the OC clock mode in Environement
	OrdinaryClockString = "OC"
	// BoundaryClockString matches the BC clock mode in Environement
	BoundaryClockString = "BC"
	// DualNICBoundaryClockString matches the DualNICBC clock mode in Environement
	DualNICBoundaryClockString = "DualNICBC"
	ptp4lEthernet              = "-2 --summary_interval -4"
	ptp4lEthernetSlave         = "-2 -s --summary_interval -4"
	phc2sysGM                  = "a -r -r -n 24" // use phc2sys to sync phc to system clock
	phc2sysSlave               = "-a -r -n 24 -m -N 8 -R 16"
	SCHED_OTHER                = "SCHED_OTHER"
	SCHED_FIFO                 = "SCHED_FIFO"
)

type ConfigStatus int64

const (
	// Start starting status when object is created
	Start ConfigStatus = iota
	// InitStatus the configuration environment variable was read
	InitStatus
	// ConfiguredStatus for OC/BC/DuallinkBC modes this is set after the ptp clock is configured
	ConfiguredStatus
	// DiscoverySuccessStatus for all modes, indicates a successful discovery
	DiscoverySuccessStatus
	// DiscoveryFailureStatus for all modes, indicates a discovery failure
	DiscoveryFailureStatus
)

type PTPMode int64

const (
	// OrdinaryClock OrdinaryClock mode
	OrdinaryClock PTPMode = iota
	// BoundaryClock Boundary Clock mode
	BoundaryClock
	// DualNICBoundaryClock DualNIC Boundary Clock mode
	DualNICBoundaryClock
	// Discovery Discovery mode
	Discovery
	// None initial empty mode
	None
)

type TestConfig struct {
	PtpModeDesired    PTPMode
	PtpModeDiscovered PTPMode
	Status            ConfigStatus
	DiscoveredGrandMasterPtpConfig,
	DiscoveredSlave1PtpConfig,
	DiscoveredSlave2PtpConfig,
	DiscoveredClockUnderTestPtpConfig,
	DiscoveredClockUnderTestSecondaryPtpConfig *ptpDiscoveryRes
	DiscoveredClockUnderTestPod *v1core.Pod
	L2Config                    l2lib.L2Info
	FoundSolutions              map[string]bool
	PtpEventsIsSidecarReady     bool
}
type solverData struct {
	// Mapping between clock role and port depending on the algo
	testClockRolesAlgoMapping map[string]*[]int
	// map storing solutions
	solutions map[string]*[][]int

	problems map[string]*[][][]int
}

var enabledProblems = []string{AlgoOCString,
	AlgoBCString,
	AlgoBCWithSlavesString,
	AlgoDualNicBCString,
	AlgoDualNicBCWithSlavesString,
	AlgoOCExtGMString,
	AlgoBCExtGMString,
	AlgoBCWithSlavesExtGMString,
	AlgoDualNicBCExtGMString,
	AlgoDualNicBCWithSlavesExtGMString,
}

const FirstSolution = 0

var data solverData

// indicates the clock roles in the algotithms
type TestIfClockRoles int

const NumTestClockRoles = 7
const (
	Grandmaster TestIfClockRoles = iota
	Slave1
	Slave2
	BC1Master
	BC1Slave
	BC2Master
	BC2Slave
)

const (
	AlgoOCString                       = "OC"
	AlgoBCString                       = "BC"
	AlgoBCWithSlavesString             = "BCWithSlaves"
	AlgoDualNicBCString                = "DualNicBC"
	AlgoDualNicBCWithSlavesString      = "DualNicBCWithSlaves"
	AlgoOCExtGMString                  = "OCExtGM"
	AlgoBCExtGMString                  = "BCExtGM"
	AlgoDualNicBCExtGMString           = "DualNicBCExtGM"
	AlgoBCWithSlavesExtGMString        = "BCWithSlavesExtGM"
	AlgoDualNicBCWithSlavesExtGMString = "DualNicBCWithSlavesExtGM"
)

type ptpDiscoveryRes ptpv1.PtpConfig

const BasePtp4lConfig = `[global]
tx_timestamp_timeout 50
ptp_dst_mac 01:1B:19:00:00:00
p2p_dst_mac 01:80:C2:00:00:0E
domainNumber 24
logging_level 7`

func (obj *ptpDiscoveryRes) String() string {
	if obj == nil {
		return "nil"
	}
	return obj.Name
}
func (obj *TestConfig) String() (out string) {
	if obj == nil {
		return "nil"
	}
	out += fmt.Sprintf("PtpModeDesired= %s, PtpModeDiscovered= %s, Status= %s, DiscoveredClockUnderTestPtpConfig= %s, DiscoveredClockUnderTestSecondaryPtpConfig= %s, DiscoveredGrandMasterPtpConfig= %s, DiscoveredSlave1PtpConfig= %s, DiscoveredSlave2PtpConfig= %s, PtpEventsIsSidecarReady= %t, ",
		obj.PtpModeDesired,
		obj.PtpModeDiscovered,
		obj.Status,
		obj.DiscoveredClockUnderTestPtpConfig,
		obj.DiscoveredClockUnderTestSecondaryPtpConfig,
		obj.DiscoveredGrandMasterPtpConfig,
		obj.DiscoveredSlave1PtpConfig,
		obj.DiscoveredSlave2PtpConfig,
		obj.PtpEventsIsSidecarReady)
	if obj.DiscoveredClockUnderTestPod != nil {
		out += fmt.Sprintf("DiscoveredClockUnderTestPodName=%s, DiscoveredClockUnderTestNodeName=%s",
			obj.DiscoveredClockUnderTestPod.Name,
			obj.DiscoveredClockUnderTestPod.Spec.NodeName)
	}

	return out
}

func (status ConfigStatus) String() string {
	switch status {
	case Start:
		return StartString
	case InitStatus:
		return InitStatusString
	case ConfiguredStatus:
		return ConfiguredStatusString
	case DiscoverySuccessStatus:
		return DiscoverySuccessStatusString
	case DiscoveryFailureStatus:
		return DiscoveryFailureStatusString
	default:
		return StartString
	}
}

func (mode PTPMode) String() string {
	switch mode {
	case OrdinaryClock:
		return OrdinaryClockString
	case BoundaryClock:
		return BoundaryClockString
	case DualNICBoundaryClock:
		return DualNICBoundaryClockString
	case Discovery:
		return DiscoveryString
	case None:
		return NoneString
	default:
		return OrdinaryClockString
	}
}

func StringToMode(aString string) PTPMode {
	switch strings.ToLower(aString) {
	case strings.ToLower(OrdinaryClockString):
		return OrdinaryClock
	case strings.ToLower(BoundaryClockString):
		return BoundaryClock
	case strings.ToLower(DualNICBoundaryClockString):
		return DualNICBoundaryClock
	case strings.ToLower(DiscoveryString), strings.ToLower(legacyDiscoveryString):
		return Discovery
	case strings.ToLower(NoneString):
		return OrdinaryClock
	default:
		return OrdinaryClock
	}
}

var GlobalConfig TestConfig

func init() {
	Reset()
}

// resets the test configuration
func Reset() {
	GlobalConfig.PtpModeDesired = None
	GlobalConfig.PtpModeDiscovered = None
	GlobalConfig.Status = Start
}
func initFoundSolutions() {
	GlobalConfig.FoundSolutions = make(map[string]bool)
	for _, name := range enabledProblems {
		if len(*data.solutions[name]) > 0 {
			GlobalConfig.FoundSolutions[name] = true
		}
	}
}

// Gets te desired configuration from the environment
func GetDesiredConfig(forceUpdate bool) TestConfig {
	defer logrus.Infof("Current PTP test config=%s", &GlobalConfig)
	if GlobalConfig.Status == InitStatus && !forceUpdate {
		return GlobalConfig
	}
	legacyDiscoveryModeString := os.Getenv("DISCOVERY_MODE")
	modeString := os.Getenv("PTP_TEST_MODE")

	mode := StringToMode(legacyDiscoveryModeString)

	if mode != Discovery {
		mode = StringToMode(modeString)
	}

	switch mode {
	case OrdinaryClock, BoundaryClock, DualNICBoundaryClock, Discovery:
		logrus.Infof("%s mode detected", mode)
		GlobalConfig.PtpModeDesired = mode
		GlobalConfig.Status = InitStatus
		return GlobalConfig
	case None:
		logrus.Infof("No test mode specified using, %s mode. Specify the env variable PTP_TEST_MODE with one of %s, %s, %s, %s", OrdinaryClock, Discovery, OrdinaryClock, BoundaryClock, DualNICBoundaryClockString)
		GlobalConfig.PtpModeDesired = OrdinaryClock
		GlobalConfig.Status = InitStatus
		return GlobalConfig
	default:
		logrus.Infof("%s is not a supported mode, assuming %s", mode, OrdinaryClock)
		GlobalConfig.PtpModeDesired = OrdinaryClock
		GlobalConfig.Status = InitStatus
		return GlobalConfig
	}
}

// Create ptpconfigs
func CreatePtpConfigurations() error {
	if GlobalConfig.PtpModeDesired != Discovery {
		// for external grand master, clean previous configuration so that it is not detected as a external grandmaster
		err := clean.All()
		if err != nil {
			logrus.Errorf("Error deleting labels and configuration, err=%s", err)
		}
		ptphelper.RestartPTPDaemon()
		ptphelper.WaitForPtpDaemonToBeReady()
	}
	// Initialize desired ptp config for all configs
	GetDesiredConfig(true)
	// in multi node configuration create ptp configs

	// Initialize l2 library
	l2lib.GlobalL2DiscoveryConfig.SetL2Client(client.Client, client.Client.Config)

	// Collect L2 info
	config, err := l2lib.GlobalL2DiscoveryConfig.GetL2DiscoveryConfig(true)
	if err != nil {
		return fmt.Errorf("Getting L2 discovery info failed with err=%s", err)
	}
	GlobalConfig.L2Config = config

	if GlobalConfig.PtpModeDesired != Discovery {
		// initialize L2 config in solver
		solver.GlobalConfig.SetL2Config(config)
		logrus.Infof("Ports getting PTP frames=%+v", config.GetPortsGettingPTP())
		initAndSolveProblems()

		if len(data.solutions) == 0 {
			return fmt.Errorf("could not find a solution")
		}
		isExternalMaster := ptphelper.IsExternalGM()
		if err != nil {
			return fmt.Errorf("cannot determine if cluster is single node")
		}
		switch GlobalConfig.PtpModeDesired {
		case Discovery, None:
			logrus.Errorf("error creating ptpconfig Discovery, None not supported")
		case OrdinaryClock:
			return PtpConfigOC(isExternalMaster)
		case BoundaryClock:
			return PtpConfigBC(isExternalMaster)
		case DualNICBoundaryClock:
			return PtpConfigDualNicBC(isExternalMaster)
		}
	}
	return nil
}

func initAndSolveProblems() {

	// create maps
	data.problems = make(map[string]*[][][]int)
	data.solutions = make(map[string]*[][]int)
	data.testClockRolesAlgoMapping = make(map[string]*[]int)

	// initialize problems
	data.problems[AlgoOCString] = &[][][]int{
		{{int(solver.StepNil), 0, 0}},         // step1
		{{int(solver.StepSameLan2), 2, 0, 1}}, // step2
	}
	data.problems[AlgoBCString] = &[][][]int{
		{{int(solver.StepNil), 0, 0}},         // step1
		{{int(solver.StepSameNic), 2, 0, 1}},  // step2
		{{int(solver.StepSameLan2), 2, 1, 2}}, // step3

	}
	data.problems[AlgoBCWithSlavesString] = &[][][]int{
		{{int(solver.StepNil), 0, 0}},         // step1
		{{int(solver.StepSameLan2), 2, 0, 1}}, // step2
		{{int(solver.StepSameNic), 2, 1, 2}},  // step3
		{{int(solver.StepSameLan2), 2, 2, 3}, // step4
			{int(solver.StepDifferentNic), 2, 0, 3}}, // step4 - downstream slaves and grandmaster must be on different nics
	}
	data.problems[AlgoDualNicBCString] = &[][][]int{
		{{int(solver.StepNil), 0, 0}},         // step1
		{{int(solver.StepSameNic), 2, 0, 1}},  // step2
		{{int(solver.StepSameLan2), 2, 1, 2}}, // step3
		{{int(solver.StepSameNode), 2, 1, 3}, // step4
			{int(solver.StepSameLan2), 2, 2, 3}}, // step4
		{{int(solver.StepSameNic), 2, 3, 4},
			{int(solver.StepDifferentNic), 2, 1, 3}}, // step5
	}
	data.problems[AlgoDualNicBCWithSlavesString] = &[][][]int{
		{{int(solver.StepNil), 0, 0}},         // step1
		{{int(solver.StepSameLan2), 2, 0, 1}}, // step2
		{{int(solver.StepSameNic), 2, 1, 2}},  // step3
		{{int(solver.StepSameLan2), 2, 2, 3}}, // step4
		{{int(solver.StepSameNode), 2, 2, 4}, // step5
			{int(solver.StepSameLan2), 2, 3, 4}}, // step5
		{{int(solver.StepSameNic), 2, 4, 5}}, // step6
		{{int(solver.StepSameLan2), 2, 5, 6}, // step7
			{int(solver.StepDifferentNic), 2, 0, 3},  // downstream slaves and grandmaster must be on different nics
			{int(solver.StepDifferentNic), 2, 6, 3},  // downstream slaves and grandmaster must be on different nics
			{int(solver.StepDifferentNic), 2, 2, 4}}, // dual nic BC uses 2 different NICs
	}
	data.problems[AlgoOCExtGMString] = &[][][]int{
		{{int(solver.StepIsPTP), 1, 0}}, // step1
	}
	data.problems[AlgoBCExtGMString] = &[][][]int{
		{{int(solver.StepIsPTP), 1, 0}},      // step1
		{{int(solver.StepSameNic), 2, 0, 1}}, // step2
	}
	data.problems[AlgoBCWithSlavesExtGMString] = &[][][]int{
		{{int(solver.StepNil), 0, 0}},         // step1
		{{int(solver.StepSameLan2), 2, 0, 1}}, // step2
		{{int(solver.StepSameNic), 2, 1, 2}, // step3
			{int(solver.StepIsPTP), 1, 2}},
	}
	data.problems[AlgoDualNicBCExtGMString] = &[][][]int{
		{{int(solver.StepIsPTP), 1, 0}},      // step1
		{{int(solver.StepSameNic), 2, 0, 1}}, // step2
		{{int(solver.StepIsPTP), 1, 2}, // step3
			{int(solver.StepSameNode), 2, 0, 2}}, // step3
		{{int(solver.StepSameNic), 2, 2, 3},
			{int(solver.StepDifferentNic), 2, 0, 2}}, // step4
	}
	data.problems[AlgoDualNicBCWithSlavesExtGMString] = &[][][]int{
		{{int(solver.StepNil), 0, 0}},         // step1
		{{int(solver.StepSameLan2), 2, 0, 1}}, // step2
		{{int(solver.StepSameNic), 2, 1, 2}},  // step3
		{{int(solver.StepSameNode), 2, 2, 4}, // step4
			{int(solver.StepIsPTP), 1, 2},
			{int(solver.StepIsPTP), 1, 4}},
		{{int(solver.StepSameNic), 2, 4, 5}}, // step5
		{{int(solver.StepSameLan2), 2, 5, 6}, // step6
			{int(solver.StepDifferentNic), 2, 0, 3},  // downstream slaves and grandmaster must be on different nics
			{int(solver.StepDifferentNic), 2, 6, 3},  // downstream slaves and grandmaster must be on different nics
			{int(solver.StepDifferentNic), 2, 2, 4}}, // dual nic BC uses 2 different NICs
	}
	// Initializing Solution decoding and mapping
	// allocating all slices
	for _, name := range enabledProblems {
		alloc := make([]int, NumTestClockRoles)
		data.testClockRolesAlgoMapping[name] = &alloc
	}

	// OC
	(*data.testClockRolesAlgoMapping[AlgoOCString])[Slave1] = 0
	(*data.testClockRolesAlgoMapping[AlgoOCString])[Grandmaster] = 1

	// BC

	(*data.testClockRolesAlgoMapping[AlgoBCString])[BC1Slave] = 0
	(*data.testClockRolesAlgoMapping[AlgoBCString])[BC1Master] = 1
	(*data.testClockRolesAlgoMapping[AlgoBCString])[Grandmaster] = 2

	// BC with slaves

	(*data.testClockRolesAlgoMapping[AlgoBCWithSlavesString])[Slave1] = 0
	(*data.testClockRolesAlgoMapping[AlgoBCWithSlavesString])[BC1Master] = 1
	(*data.testClockRolesAlgoMapping[AlgoBCWithSlavesString])[BC1Slave] = 2
	(*data.testClockRolesAlgoMapping[AlgoBCWithSlavesString])[Grandmaster] = 3

	// Dual NIC BC
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCString])[BC1Slave] = 0
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCString])[BC1Master] = 1
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCString])[Grandmaster] = 2
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCString])[BC2Master] = 3
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCString])[BC2Slave] = 4

	// Dual NIC BC with slaves

	(*data.testClockRolesAlgoMapping[AlgoDualNicBCWithSlavesString])[Slave1] = 0
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCWithSlavesString])[BC1Master] = 1
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCWithSlavesString])[BC1Slave] = 2
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCWithSlavesString])[Grandmaster] = 3
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCWithSlavesString])[BC2Slave] = 4
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCWithSlavesString])[BC2Master] = 5
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCWithSlavesString])[Slave2] = 6

	// OC, External GM
	(*data.testClockRolesAlgoMapping[AlgoOCExtGMString])[Slave1] = 0

	// BC, External GM
	(*data.testClockRolesAlgoMapping[AlgoBCExtGMString])[BC1Slave] = 0
	(*data.testClockRolesAlgoMapping[AlgoBCExtGMString])[BC1Master] = 1

	// BC with slaves, External GM

	(*data.testClockRolesAlgoMapping[AlgoBCWithSlavesExtGMString])[Slave1] = 0
	(*data.testClockRolesAlgoMapping[AlgoBCWithSlavesExtGMString])[BC1Master] = 1
	(*data.testClockRolesAlgoMapping[AlgoBCWithSlavesExtGMString])[BC1Slave] = 2

	// Dual NIC BC, External GM

	(*data.testClockRolesAlgoMapping[AlgoDualNicBCExtGMString])[BC1Slave] = 0
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCExtGMString])[BC1Master] = 1
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCExtGMString])[BC2Slave] = 2
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCExtGMString])[BC2Master] = 3

	// Dual NIC BC with slaves, External GM
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCWithSlavesExtGMString])[Slave1] = 0
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCWithSlavesExtGMString])[BC1Master] = 1
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCWithSlavesExtGMString])[BC1Slave] = 2
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCWithSlavesExtGMString])[BC2Slave] = 4
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCWithSlavesExtGMString])[BC2Master] = 5
	(*data.testClockRolesAlgoMapping[AlgoDualNicBCWithSlavesExtGMString])[Slave2] = 6

	for _, name := range enabledProblems {
		// Initializing problems
		solver.GlobalConfig.InitProblem(
			name,
			*data.problems[name],
			*data.testClockRolesAlgoMapping[name],
		)

		// Solve problem
		solver.GlobalConfig.Run(name)
	}

	// print first solution
	solver.GlobalConfig.PrintFirstSolution()

	// store the solutions
	data.solutions = solver.GlobalConfig.GetSolutions()

	// update testconfig found solutions
	initFoundSolutions()
}

// Gets the discovered configuration
func GetFullDiscoveredConfig(namespace string, forceUpdate bool) TestConfig {
	logrus.Infof("Getting ptp configuration for namespace:%s", namespace)
	defer logrus.Infof("Current PTP test config=%s", &GlobalConfig)

	if GlobalConfig.Status == DiscoveryFailureStatus ||
		GlobalConfig.Status == DiscoverySuccessStatus && !forceUpdate {
		return GlobalConfig
	}

	discoverPTPConfiguration(namespace)
	return GlobalConfig
}

func CreatePtpConfigGrandMaster(nodeName, ifName string) error {
	ptpSchedulingPolicy := SCHED_OTHER
	configureFifo, err := strconv.ParseBool(os.Getenv("CONFIGURE_FIFO"))
	if err == nil && configureFifo {
		ptpSchedulingPolicy = SCHED_FIFO
	}
	// Labeling the grandmaster node
	_, err = nodes.LabelNode(nodeName, pkg.PtpGrandmasterNodeLabel, "")
	if err != nil {
		logrus.Errorf("Error setting Grandmaster node role label: %s", err)
	}

	// Grandmaster
	gmConfig := BasePtp4lConfig + "\nmasterOnly 1"
	ptp4lsysOpts := ptp4lEthernet
	phc2sysOpts := phc2sysGM
	return createConfig(pkg.PtpGrandMasterPolicyName,
		&ifName,
		&ptp4lsysOpts,
		gmConfig,
		&phc2sysOpts,
		pkg.PtpGrandmasterNodeLabel,
		pointer.Int64Ptr(int5),
		ptpSchedulingPolicy,
		pointer.Int64Ptr(int65))
}

func CreatePtpConfigBC(policyName, nodeName, ifMasterName, ifSlaveName string, phc2sys bool) (err error) {
	ptpSchedulingPolicy := SCHED_OTHER
	configureFifo, err := strconv.ParseBool(os.Getenv("CONFIGURE_FIFO"))
	if err == nil && configureFifo {
		ptpSchedulingPolicy = SCHED_FIFO
	}
	// Sleep for a second to allow previous label on the same node to complete
	time.Sleep(time.Second)
	_, err = nodes.LabelNode(nodeName, pkg.PtpClockUnderTestNodeLabel, "")
	if err != nil {
		logrus.Errorf("Error setting BC node role label: %s", err)
	}

	bcConfig := BasePtp4lConfig + "\nboundary_clock_jbod 1\ngmCapable 0"
	bcConfig = AddInterface(bcConfig, ifSlaveName, 0)
	bcConfig = AddInterface(bcConfig, ifMasterName, 1)
	ptp4lsysOpts := ptp4lEthernet

	var phc2sysOpts *string
	temp := phc2sysSlave
	if phc2sys {
		phc2sysOpts = &temp
	} else {
		phc2sysOpts = nil
	}
	return createConfig(policyName,
		nil,
		&ptp4lsysOpts,
		bcConfig,
		phc2sysOpts,
		pkg.PtpClockUnderTestNodeLabel,
		pointer.Int64Ptr(int5),
		ptpSchedulingPolicy,
		pointer.Int64Ptr(int65))
}

func CreatePtpConfigOC(profileName, nodeName, ifSlaveName string, phc2sys bool, label string) (err error) {
	ptpSchedulingPolicy := SCHED_OTHER
	configureFifo, err := strconv.ParseBool(os.Getenv("CONFIGURE_FIFO"))
	if err == nil && configureFifo {
		ptpSchedulingPolicy = SCHED_FIFO
	}
	// Sleep for a second to allow previous label on the same node to complete
	time.Sleep(time.Second)
	_, err = nodes.LabelNode(nodeName, label, "")
	if err != nil {
		logrus.Errorf("Error setting Slave node role label: %s", err)
	}
	ptp4lsysOpts := ptp4lEthernetSlave
	var phc2sysOpts *string
	temp := phc2sysSlave
	if phc2sys {
		phc2sysOpts = &temp
	} else {
		phc2sysOpts = nil
	}

	return createConfig(profileName,
		&ifSlaveName,
		&ptp4lsysOpts,
		BasePtp4lConfig,
		phc2sysOpts,
		label,
		pointer.Int64Ptr(int5),
		ptpSchedulingPolicy,
		pointer.Int64Ptr(int65))
}

func PtpConfigOC(isExtGM bool) error {
	var grandmaster, slave1 int

	BestSolution := ""

	if isExtGM {
		if len(*data.solutions[AlgoOCExtGMString]) != 0 {
			BestSolution = AlgoOCExtGMString
		} else {
			return fmt.Errorf("no solution found for OC configuration in External GM mode")
		}
	} else {
		if len(*data.solutions[AlgoOCString]) != 0 {
			BestSolution = AlgoOCString
		}
		if BestSolution == "" {
			return fmt.Errorf("no solution found for OC configuration in Local GM mode")
		}
	}
	logrus.Infof("Configuring best solution= %s", BestSolution)
	switch BestSolution {
	case AlgoOCString:
		grandmaster = (*data.testClockRolesAlgoMapping[BestSolution])[Grandmaster]
		slave1 = (*data.testClockRolesAlgoMapping[BestSolution])[Slave1]

		gmIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][grandmaster]]
		slave1If := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][slave1]]

		err := CreatePtpConfigGrandMaster(gmIf.NodeName,
			gmIf.IfName)
		if err != nil {
			logrus.Errorf("Error creating Grandmaster ptpconfig: %s", err)
		}

		err = CreatePtpConfigOC(pkg.PtpSlave1PolicyName, slave1If.NodeName,
			slave1If.IfName, true, pkg.PtpClockUnderTestNodeLabel)
		if err != nil {
			logrus.Errorf("Error creating Slave1 ptpconfig: %s", err)
		}

	case AlgoOCExtGMString:

		slave1 = (*data.testClockRolesAlgoMapping[BestSolution])[Slave1]

		slave1If := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][slave1]]

		err := CreatePtpConfigOC(pkg.PtpSlave1PolicyName, slave1If.NodeName,
			slave1If.IfName, true, pkg.PtpClockUnderTestNodeLabel)
		if err != nil {
			logrus.Errorf("Error creating Slave1 ptpconfig: %s", err)
		}
	}
	return nil
}
func PtpConfigBC(isExtGM bool) error {
	var grandmaster, bc1Master, bc1Slave, slave1 int

	BestSolution := ""

	if isExtGM {
		if len(*data.solutions[AlgoBCExtGMString]) != 0 {
			BestSolution = AlgoBCExtGMString
		}
		if len(*data.solutions[AlgoBCWithSlavesExtGMString]) != 0 {
			BestSolution = AlgoBCWithSlavesExtGMString
		}
		if BestSolution == "" {
			return fmt.Errorf("no solution found for BC configuration in External GM mode")
		}

	} else {
		if len(*data.solutions[AlgoBCString]) != 0 {
			BestSolution = AlgoBCString
		}
		if len(*data.solutions[AlgoBCWithSlavesString]) != 0 {
			BestSolution = AlgoBCWithSlavesString
		}
		if BestSolution == "" {
			return fmt.Errorf("no solution found for BC configuration in Local GM mode")
		}
	}

	logrus.Infof("Configuring best solution= %s", BestSolution)
	switch BestSolution {
	case AlgoBCWithSlavesString:
		grandmaster = (*data.testClockRolesAlgoMapping[BestSolution])[Grandmaster]
		bc1Master = (*data.testClockRolesAlgoMapping[BestSolution])[BC1Master]
		bc1Slave = (*data.testClockRolesAlgoMapping[BestSolution])[BC1Slave]
		slave1 = (*data.testClockRolesAlgoMapping[BestSolution])[Slave1]

		gmIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][grandmaster]]
		bc1MasterIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc1Master]]
		bc1SlaveIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc1Slave]]
		slave1If := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][slave1]]

		err := CreatePtpConfigGrandMaster(gmIf.NodeName,
			gmIf.IfName)
		if err != nil {
			logrus.Errorf("Error creating Grandmaster ptpconfig: %s", err)
		}

		err = CreatePtpConfigBC(pkg.PtpBcMaster1PolicyName, bc1MasterIf.NodeName,
			bc1MasterIf.IfName, bc1SlaveIf.IfName, true)
		if err != nil {
			logrus.Errorf("Error creating bc1master ptpconfig: %s", err)
		}

		err = CreatePtpConfigOC(pkg.PtpSlave1PolicyName, slave1If.NodeName,
			slave1If.IfName, false, pkg.PtpSlave1NodeLabel)
		if err != nil {
			logrus.Errorf("Error creating Slave1 ptpconfig: %s", err)
		}

	case AlgoBCString:
		grandmaster = (*data.testClockRolesAlgoMapping[BestSolution])[Grandmaster]
		bc1Master = (*data.testClockRolesAlgoMapping[BestSolution])[BC1Master]
		bc1Slave = (*data.testClockRolesAlgoMapping[BestSolution])[BC1Slave]

		gmIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][grandmaster]]
		bc1MasterIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc1Master]]
		bc1SlaveIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc1Slave]]

		err := CreatePtpConfigGrandMaster(gmIf.NodeName,
			gmIf.IfName)
		if err != nil {
			logrus.Errorf("Error creating Grandmaster ptpconfig: %s", err)
		}

		err = CreatePtpConfigBC(pkg.PtpBcMaster1PolicyName, bc1MasterIf.NodeName,
			bc1MasterIf.IfName, bc1SlaveIf.IfName, true)
		if err != nil {
			logrus.Errorf("Error creating bc1master ptpconfig: %s", err)
		}

	case AlgoBCExtGMString:

		bc1Master = (*data.testClockRolesAlgoMapping[BestSolution])[BC1Master]
		bc1Slave = (*data.testClockRolesAlgoMapping[BestSolution])[BC1Slave]

		bc1MasterIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc1Master]]
		bc1SlaveIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc1Slave]]

		err := CreatePtpConfigBC(pkg.PtpBcMaster1PolicyName, bc1MasterIf.NodeName,
			bc1MasterIf.IfName, bc1SlaveIf.IfName, true)
		if err != nil {
			logrus.Errorf("Error creating bc1master ptpconfig: %s", err)
		}
	case AlgoBCWithSlavesExtGMString:
		bc1Master = (*data.testClockRolesAlgoMapping[BestSolution])[BC1Master]
		bc1Slave = (*data.testClockRolesAlgoMapping[BestSolution])[BC1Slave]
		slave1 = (*data.testClockRolesAlgoMapping[BestSolution])[Slave1]

		bc1MasterIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc1Master]]
		bc1SlaveIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc1Slave]]
		slave1If := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][slave1]]

		err := CreatePtpConfigBC(pkg.PtpBcMaster1PolicyName, bc1MasterIf.NodeName,
			bc1MasterIf.IfName, bc1SlaveIf.IfName, true)
		if err != nil {
			logrus.Errorf("Error creating bc1master ptpconfig: %s", err)
		}

		err = CreatePtpConfigOC(pkg.PtpSlave1PolicyName, slave1If.NodeName,
			slave1If.IfName, false, pkg.PtpSlave1NodeLabel)
		if err != nil {
			logrus.Errorf("Error creating Slave1 ptpconfig: %s", err)
		}
	}
	return nil
}

func PtpConfigDualNicBC(isExtGM bool) error {

	var grandmaster, bc1Master, bc1Slave, slave1, bc2Master, bc2Slave, slave2 int

	BestSolution := ""
	if isExtGM {
		if len(*data.solutions[AlgoDualNicBCExtGMString]) != 0 {
			BestSolution = AlgoDualNicBCExtGMString
		}
		if len(*data.solutions[AlgoDualNicBCWithSlavesExtGMString]) != 0 {
			BestSolution = AlgoDualNicBCWithSlavesExtGMString
		}
		if BestSolution == "" {
			return fmt.Errorf("no solution found for Dual NIC BC configuration in External GM mode")
		}
	} else {
		if len(*data.solutions[AlgoDualNicBCString]) != 0 {
			BestSolution = AlgoDualNicBCString
		}
		if len(*data.solutions[AlgoDualNicBCWithSlavesString]) != 0 {
			BestSolution = AlgoDualNicBCWithSlavesString
		}
		if BestSolution == "" {
			return fmt.Errorf("no solution found for Dual NIC BC configuration in Local GM mode")
		}
	}

	logrus.Infof("Configuring best solution= %s", BestSolution)
	switch BestSolution {
	case AlgoDualNicBCWithSlavesString:
		grandmaster = (*data.testClockRolesAlgoMapping[BestSolution])[Grandmaster]
		bc1Master = (*data.testClockRolesAlgoMapping[BestSolution])[BC1Master]
		bc1Slave = (*data.testClockRolesAlgoMapping[BestSolution])[BC1Slave]
		slave1 = (*data.testClockRolesAlgoMapping[BestSolution])[Slave1]
		bc2Master = (*data.testClockRolesAlgoMapping[BestSolution])[BC2Master]
		bc2Slave = (*data.testClockRolesAlgoMapping[BestSolution])[BC2Slave]
		slave2 = (*data.testClockRolesAlgoMapping[BestSolution])[Slave2]

		gmIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][grandmaster]]
		bc1MasterIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc1Master]]
		bc1SlaveIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc1Slave]]
		slave1If := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][slave1]]
		bc2MasterIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc2Master]]
		bc2SlaveIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc2Slave]]
		slave2If := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][slave2]]

		err := CreatePtpConfigGrandMaster(gmIf.NodeName,
			gmIf.IfName)
		if err != nil {
			logrus.Errorf("Error creating Grandmaster ptpconfig: %s", err)
		}

		err = CreatePtpConfigBC(pkg.PtpBcMaster1PolicyName, bc1MasterIf.NodeName,
			bc1MasterIf.IfName, bc1SlaveIf.IfName, true)
		if err != nil {
			logrus.Errorf("Error creating bc1master ptpconfig: %s", err)
		}

		err = CreatePtpConfigBC(pkg.PtpBcMaster2PolicyName, bc2MasterIf.NodeName,
			bc2MasterIf.IfName, bc2SlaveIf.IfName, false)
		if err != nil {
			logrus.Errorf("Error creating bc2master ptpconfig: %s", err)
		}

		err = CreatePtpConfigOC(pkg.PtpSlave1PolicyName, slave1If.NodeName,
			slave1If.IfName, false, pkg.PtpSlave1NodeLabel)
		if err != nil {
			logrus.Errorf("Error creating Slave1 ptpconfig: %s", err)
		}
		err = CreatePtpConfigOC(pkg.PtpSlave2PolicyName, slave2If.NodeName,
			slave2If.IfName, false, pkg.PtpSlave2NodeLabel)
		if err != nil {
			logrus.Errorf("Error creating Slave2 ptpconfig: %s", err)
		}

	case AlgoDualNicBCString:
		grandmaster = (*data.testClockRolesAlgoMapping[BestSolution])[Grandmaster]
		bc1Master = (*data.testClockRolesAlgoMapping[BestSolution])[BC1Master]
		bc1Slave = (*data.testClockRolesAlgoMapping[BestSolution])[BC1Slave]
		bc2Master = (*data.testClockRolesAlgoMapping[BestSolution])[BC2Master]
		bc2Slave = (*data.testClockRolesAlgoMapping[BestSolution])[BC2Slave]

		gmIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][grandmaster]]
		bc1MasterIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc1Master]]
		bc1SlaveIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc1Slave]]
		bc2MasterIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc2Master]]
		bc2SlaveIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc2Slave]]

		err := CreatePtpConfigGrandMaster(gmIf.NodeName,
			gmIf.IfName)
		if err != nil {
			logrus.Errorf("Error creating Grandmaster ptpconfig: %s", err)
		}

		err = CreatePtpConfigBC(pkg.PtpBcMaster1PolicyName, bc1MasterIf.NodeName,
			bc1MasterIf.IfName, bc1SlaveIf.IfName, true)
		if err != nil {
			logrus.Errorf("Error creating bc1master ptpconfig: %s", err)
		}

		err = CreatePtpConfigBC(pkg.PtpBcMaster2PolicyName, bc2MasterIf.NodeName,
			bc2MasterIf.IfName, bc2SlaveIf.IfName, false)
		if err != nil {
			logrus.Errorf("Error creating bc2master ptpconfig: %s", err)
		}

	case AlgoDualNicBCExtGMString:

		bc1Master = (*data.testClockRolesAlgoMapping[BestSolution])[BC1Master]
		bc1Slave = (*data.testClockRolesAlgoMapping[BestSolution])[BC1Slave]
		bc2Master = (*data.testClockRolesAlgoMapping[BestSolution])[BC2Master]
		bc2Slave = (*data.testClockRolesAlgoMapping[BestSolution])[BC2Slave]

		bc1MasterIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc1Master]]
		bc1SlaveIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc1Slave]]
		bc2MasterIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc2Master]]
		bc2SlaveIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc2Slave]]

		err := CreatePtpConfigBC(pkg.PtpBcMaster1PolicyName, bc1MasterIf.NodeName,
			bc1MasterIf.IfName, bc1SlaveIf.IfName, true)
		if err != nil {
			logrus.Errorf("Error creating bc1master ptpconfig: %s", err)
		}

		err = CreatePtpConfigBC(pkg.PtpBcMaster2PolicyName, bc2MasterIf.NodeName,
			bc2MasterIf.IfName, bc2SlaveIf.IfName, false)
		if err != nil {
			logrus.Errorf("Error creating bc2master ptpconfig: %s", err)
		}

	case AlgoDualNicBCWithSlavesExtGMString:
		bc1Master = (*data.testClockRolesAlgoMapping[BestSolution])[BC1Master]
		bc1Slave = (*data.testClockRolesAlgoMapping[BestSolution])[BC1Slave]
		slave1 = (*data.testClockRolesAlgoMapping[BestSolution])[Slave1]
		bc2Master = (*data.testClockRolesAlgoMapping[BestSolution])[BC2Master]
		bc2Slave = (*data.testClockRolesAlgoMapping[BestSolution])[BC2Slave]
		slave2 = (*data.testClockRolesAlgoMapping[BestSolution])[Slave2]

		bc1MasterIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc1Master]]
		bc1SlaveIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc1Slave]]
		slave1If := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][slave1]]
		bc2MasterIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc2Master]]
		bc2SlaveIf := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][bc2Slave]]
		slave2If := GlobalConfig.L2Config.GetPtpIfList()[(*data.solutions[BestSolution])[FirstSolution][slave2]]

		err := CreatePtpConfigBC(pkg.PtpBcMaster1PolicyName, bc1MasterIf.NodeName,
			bc1MasterIf.IfName, bc1SlaveIf.IfName, true)
		if err != nil {
			logrus.Errorf("Error creating bc1master ptpconfig: %s", err)
		}

		err = CreatePtpConfigBC(pkg.PtpBcMaster2PolicyName, bc2MasterIf.NodeName,
			bc2MasterIf.IfName, bc2SlaveIf.IfName, false)
		if err != nil {
			logrus.Errorf("Error creating bc2master ptpconfig: %s", err)
		}

		err = CreatePtpConfigOC(pkg.PtpSlave1PolicyName, slave1If.NodeName,
			slave1If.IfName, false, pkg.PtpSlave1NodeLabel)
		if err != nil {
			logrus.Errorf("Error creating Slave1 ptpconfig: %s", err)
		}
		err = CreatePtpConfigOC(pkg.PtpSlave2PolicyName, slave2If.NodeName,
			slave2If.IfName, false, pkg.PtpSlave2NodeLabel)
		if err != nil {
			logrus.Errorf("Error creating Slave2 ptpconfig: %s", err)
		}
	}
	return nil
}

// helper function to add an interface to the ptp4l config
func AddInterface(ptpConfig, iface string, masterOnly int) (updatedPtpConfig string) {
	return fmt.Sprintf("%s\n[%s]\nmasterOnly %d", ptpConfig, iface, masterOnly)
}

// helper function to create a ptpconfig
func createConfig(profileName string, ifaceName, ptp4lOpts *string, ptp4lConfig string, phc2sysOpts *string, nodeLabel string, priority *int64, ptpSchedulingPolicy string, ptpSchedulingPriority *int64) error {
	thresholds := ptpv1.PtpClockThreshold{}

	testParameters, err := ptptestconfig.GetPtpTestConfig()
	if err != nil {
		return fmt.Errorf("failed to get test config: %v", err)
	}
	thresholds.MaxOffsetThreshold = int64(testParameters.GlobalConfig.MaxOffset)
	thresholds.MinOffsetThreshold = int64(testParameters.GlobalConfig.MinOffset)
	ptpProfile := ptpv1.PtpProfile{Name: &profileName, Interface: ifaceName, Phc2sysOpts: phc2sysOpts, Ptp4lOpts: ptp4lOpts, PtpSchedulingPolicy: &ptpSchedulingPolicy, PtpSchedulingPriority: ptpSchedulingPriority,
		PtpClockThreshold: &thresholds}
	if ptp4lConfig != "" {
		ptpProfile.Ptp4lConf = &ptp4lConfig
	}
	matchRule := ptpv1.MatchRule{NodeLabel: &nodeLabel}
	ptpRecommend := ptpv1.PtpRecommend{Profile: &profileName, Priority: priority, Match: []ptpv1.MatchRule{matchRule}}

	policy := ptpv1.PtpConfig{ObjectMeta: metav1.ObjectMeta{Name: profileName, Namespace: PtpLinuxDaemonNamespace},
		Spec: ptpv1.PtpConfigSpec{Profile: []ptpv1.PtpProfile{ptpProfile},
			Recommend: []ptpv1.PtpRecommend{ptpRecommend}}}

	_, err = client.Client.PtpConfigs(PtpLinuxDaemonNamespace).Create(context.Background(), &policy, metav1.CreateOptions{})
	return err
}

// Discovers the PTP configuration
func discoverPTPConfiguration(namespace string) {
	var ptpConfigClockUnderTest []*ptpv1.PtpConfig

	configList, err := client.Client.PtpConfigs(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logrus.Errorf("error getting ptpconfig list, err=%s", err)
	}
	logrus.Infof("%d ptpconfig objects recovered", len(configList.Items))
	for profileIndex := range configList.Items {
		for _, r := range configList.Items[profileIndex].Spec.Recommend {
			for _, m := range r.Match {
				if m.NodeLabel == nil {
					continue
				}
				if *m.NodeLabel == pkg.PtpClockUnderTestNodeLabel {
					ptpConfigClockUnderTest = append(ptpConfigClockUnderTest, &configList.Items[profileIndex])
				}

				// Grand master, slave 1 and slave 2 are checked as they are always created by the test program
				if GlobalConfig.PtpModeDesired != Discovery && GlobalConfig.PtpModeDesired != None {
					if *m.NodeLabel == pkg.PtpGrandmasterNodeLabel {
						GlobalConfig.DiscoveredGrandMasterPtpConfig = (*ptpDiscoveryRes)(&configList.Items[profileIndex])
					}
					if *m.NodeLabel == pkg.PtpSlave1NodeLabel {
						GlobalConfig.DiscoveredSlave1PtpConfig = (*ptpDiscoveryRes)(&configList.Items[profileIndex])
					}
					if *m.NodeLabel == pkg.PtpSlave2NodeLabel {
						GlobalConfig.DiscoveredSlave2PtpConfig = (*ptpDiscoveryRes)(&configList.Items[profileIndex])
					}
				}
			}
		}
	}
	discoverMode(ptpConfigClockUnderTest)
}

// Helper function analysing ptpconfig to deduce the actual ptp configuration
func discoverMode(ptpConfigClockUnderTest []*ptpv1.PtpConfig) {
	GlobalConfig.Status = DiscoveryFailureStatus
	if len(ptpConfigClockUnderTest) == 0 {
		logrus.Warnf("No Configs present, cannot discover")
		return
	}
	numBc := 0
	numSecondaryBC := 0

	GlobalConfig.Status = DiscoveryFailureStatus

	for _, ptpConfig := range ptpConfigClockUnderTest {
		masterIf := len(ptpv1.GetInterfaces(*ptpConfig, ptpv1.Master))
		slaveIf := len(ptpv1.GetInterfaces(*ptpConfig, ptpv1.Slave))
		// OC
		if masterIf == 0 && slaveIf == 1 && len(ptpConfigClockUnderTest) == 1 {
			GlobalConfig.PtpModeDiscovered = OrdinaryClock
			GlobalConfig.Status = DiscoverySuccessStatus
			GlobalConfig.DiscoveredClockUnderTestPtpConfig = (*ptpDiscoveryRes)(ptpConfig)
			break
		}
		// BC and Dual NIC BC
		if masterIf >= 1 && slaveIf >= 1 {
			if numBc == 0 {
				GlobalConfig.DiscoveredClockUnderTestPtpConfig = (*ptpDiscoveryRes)(ptpConfig)
			}
			if numBc == 1 {
				GlobalConfig.DiscoveredClockUnderTestSecondaryPtpConfig = (*ptpDiscoveryRes)(ptpConfig)
			}
			numBc++
			if ptphelper.IsSecondaryBc(ptpConfig) {
				numSecondaryBC++
			}
		}
	}
	if numBc == 1 {
		GlobalConfig.PtpModeDiscovered = BoundaryClock
		GlobalConfig.Status = DiscoverySuccessStatus
	}
	if numBc == 2 && numSecondaryBC == 1 {
		GlobalConfig.PtpModeDiscovered = DualNICBoundaryClock
		GlobalConfig.Status = DiscoverySuccessStatus
	}
	pod, err := ptphelper.GetPTPPodWithPTPConfig((*ptpv1.PtpConfig)(GlobalConfig.DiscoveredClockUnderTestPtpConfig))
	if err != nil {
		logrus.Error("Could not determine ptp daemon pod selected by ptpconfig")
	}
	GlobalConfig.DiscoveredClockUnderTestPod = pod
}
