package l2lib

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/sirupsen/logrus"
	l2 "github.com/test-network-function/l2discovery-exports"
	"github.com/test-network-function/l2discovery-lib/pkg/l2client"
	"github.com/test-network-function/l2discovery-lib/pkg/pods"
	daemonsets "github.com/test-network-function/privileged-daemonset"
	"github.com/yourbasic/graph"
	v1core "k8s.io/api/core/v1"
)

func init() {
	GlobalL2DiscoveryConfig.refresh = true
}

type L2Info interface {
	// list of cluster interfaces indexed with a simple integer (X) for readability in the graph
	GetPtpIfList() []*l2.PtpIf
	// LANs identified in the graph
	GetLANs() *[][]int
	// List of port receiving PTP frames (assuming valid GM signal received)
	GetPortsGettingPTP() []*l2.PtpIf

	SetL2Client(kubernetes.Interface, *rest.Config)
	GetL2DiscoveryConfig(ptpInterfacesOnly bool) (config L2Info, err error)
}

const (
	ExperimentalEthertype          = "88b5"
	PtpEthertype                   = "88f7"
	LocalInterfaces                = "0000"
	L2DaemonsetManagedString       = "MANAGED"
	L2DaemonsetPreConfiguredString = "PRECONFIGURED"
	L2DiscoveryDsName              = "l2discovery"
	L2DiscoveryNsName              = "default"
	L2DiscoveryContainerName       = "l2discovery"
	timeoutDaemon                  = time.Second * 60
	L2DiscoveryDuration            = time.Second * 15
	l2DiscoveryImage               = "quay.io/testnetworkfunction/l2discovery:v5"
)

type L2DaemonsetMode int64

const (
	// In managed mode, the L2 Topology discovery Daemonset is created by the conformance suite
	Managed L2DaemonsetMode = iota
	// In pre-configured mode, the L2 topology daemonset is pre-configured by the user in the cluster
	PreConfigured
)

func (mode L2DaemonsetMode) String() string {
	switch mode {
	case Managed:
		return L2DaemonsetManagedString
	case PreConfigured:
		return L2DaemonsetPreConfiguredString
	default:
		return L2DaemonsetManagedString
	}
}

func StringToL2Mode(aString string) L2DaemonsetMode {
	switch aString {
	case L2DaemonsetManagedString:
		return Managed
	case L2DaemonsetPreConfiguredString:
		return PreConfigured
	default:
		return Managed
	}
}

type L2DiscoveryConfig struct {
	// Map of L2 topology as discovered by L2 discovery mechanism
	DiscoveryMap map[string]map[string]map[string]*l2.Neighbors
	// L2 topology graph created from discovery map. This is the main internal graph
	L2ConnectivityMap *graph.Mutable
	// Max size of graph
	MaxL2GraphSize int
	// list of cluster interfaces indexed with a simple integer (X) for readability in the graph
	PtpIfList []*l2.PtpIf
	// list of L2discovery daemonset pods
	L2DiscoveryPods map[string]*v1core.Pod
	// Mapping between clusterwide interface index and Mac address
	ClusterMacs map[l2.IfClusterIndex]string
	// Mapping between clusterwide interface index and a simple integer (X) for readability in the graph
	ClusterIndexToInt map[l2.IfClusterIndex]int
	// Mapping between a cluster wide MAC address and a simple integer (X) for readability in the graph
	ClusterMacToInt map[string]int
	// Mapping between a Mac address and a cluster wide interface index
	ClusterIndexes map[string]l2.IfClusterIndex
	// indicates whether the L2discovery daemonset is created by the test suite (managed) or not
	L2DsMode L2DaemonsetMode
	// LANs identified in the graph
	LANs *[][]int
	// List of port receiving PTP frames (assuming valid GM signal received)
	PortsGettingPTP []*l2.PtpIf
	// interfaces to avoid when running the tests
	SkippedInterfaces []string
	// Indicates that the L2 configuration must be refreshed
	refresh bool
}

var GlobalL2DiscoveryConfig L2DiscoveryConfig

func (config *L2DiscoveryConfig) GetPtpIfList() []*l2.PtpIf {
	return config.PtpIfList
}
func (config *L2DiscoveryConfig) GetLANs() *[][]int {
	return config.LANs
}
func (config *L2DiscoveryConfig) GetPortsGettingPTP() []*l2.PtpIf {
	return config.PortsGettingPTP
}
func (config *L2DiscoveryConfig) SetL2Client(k8sClient kubernetes.Interface, restClient *rest.Config) {
	l2client.Set(k8sClient, restClient)
}

// Gets existing L2 configuration or creates a new one  (if refresh is set to true)
func (config *L2DiscoveryConfig) GetL2DiscoveryConfig(ptpInterfacesOnly bool) (L2Info, error) {
	if GlobalL2DiscoveryConfig.refresh {
		err := GlobalL2DiscoveryConfig.DiscoverL2Connectivity(ptpInterfacesOnly)
		if err != nil {
			GlobalL2DiscoveryConfig.refresh = false
			return nil, fmt.Errorf("failed to discover L2 connectivity: %w", err)
		}
	}
	GlobalL2DiscoveryConfig.refresh = false
	return &GlobalL2DiscoveryConfig, nil
}

// Resets the L2 configuration
func (config *L2DiscoveryConfig) reset() {
	GlobalL2DiscoveryConfig.PtpIfList = []*l2.PtpIf{}
	GlobalL2DiscoveryConfig.L2DiscoveryPods = make(map[string]*v1core.Pod)
	GlobalL2DiscoveryConfig.ClusterMacs = make(map[l2.IfClusterIndex]string)
	GlobalL2DiscoveryConfig.ClusterIndexes = make(map[string]l2.IfClusterIndex)
	GlobalL2DiscoveryConfig.ClusterMacToInt = make(map[string]int)
	GlobalL2DiscoveryConfig.ClusterIndexToInt = make(map[l2.IfClusterIndex]int)
	GlobalL2DiscoveryConfig.ClusterIndexes = make(map[string]l2.IfClusterIndex)
}

// Discovers the L2 connectivity using l2discovery daemonset
func (config *L2DiscoveryConfig) DiscoverL2Connectivity(ptpInterfacesOnly bool) error {
	GlobalL2DiscoveryConfig.reset()

	// initializes clusterwide ptp interfaces
	var err error
	// Create L2 discovery daemonset
	config.L2DsMode = StringToL2Mode(os.Getenv("L2_DAEMONSET"))
	if config.L2DsMode == Managed {
		_, err = daemonsets.CreateDaemonSet(L2DiscoveryDsName, L2DiscoveryNsName, L2DiscoveryContainerName, l2DiscoveryImage, timeoutDaemon)
		if err != nil {
			return fmt.Errorf("error creating l2 discovery daemonset, err=%s", err)
		}
	}
	// Sleep a short time to allow discovery to happen (first report after 5s)
	time.Sleep(L2DiscoveryDuration)
	// Get the L2 topology pods
	err = GlobalL2DiscoveryConfig.getL2TopologyDiscoveryPods()
	if err != nil {
		return fmt.Errorf("could not get l2 discovery pods, err=%s", err)
	}
	err = config.getL2Disc(ptpInterfacesOnly)
	if err != nil {
		logrus.Errorf("error getting l2 discovery data, err=%s", err)
	}
	// Delete L2 discovery daemonset
	if config.L2DsMode == Managed {
		err = daemonsets.DeleteDaemonSet(L2DiscoveryDsName, L2DiscoveryNsName)
		if err != nil {
			logrus.Errorf("error deleting l2 discovery daemonset, err=%s", err)
		}
	}
	// Create a graph from the discovered data
	err = config.createL2InternalGraph(ptpInterfacesOnly)
	if err != nil {
		return err
	}
	return nil
}

// Print database with all NICs
func (config *L2DiscoveryConfig) PrintAllNICs() {
	for index, aIf := range config.PtpIfList {
		logrus.Infof("%d %s", index, aIf)
	}

	for index, island := range *config.LANs {
		aLog := fmt.Sprintf("island %d: ", index)
		for _, aIf := range island {
			aLog += fmt.Sprintf("%s **** ", config.PtpIfList[aIf])
		}
		logrus.Info(aLog)
	}
}

// Gets the latest topology reports from the l2discovery pods
func (config *L2DiscoveryConfig) getL2Disc(ptpInterfacesOnly bool) error {
	config.DiscoveryMap = make(map[string]map[string]map[string]*l2.Neighbors)
	index := 0
	for _, aPod := range config.L2DiscoveryPods {
		podLogs, _ := pods.GetLog(aPod, aPod.Spec.Containers[0].Name)
		indexReport := strings.LastIndex(podLogs, "JSON_REPORT")
		report := strings.Split(strings.Split(podLogs[indexReport:], `\n`)[0], "JSON_REPORT")[1]
		var discDataPerNode map[string]map[string]*l2.Neighbors
		if err := json.Unmarshal([]byte(report), &discDataPerNode); err != nil {
			return err
		}

		if _, ok := config.DiscoveryMap[aPod.Spec.NodeName]; !ok {
			config.DiscoveryMap[aPod.Spec.NodeName] = make(map[string]map[string]*l2.Neighbors)
		}
		config.DiscoveryMap[aPod.Spec.NodeName] = discDataPerNode

		config.createMaps(discDataPerNode, aPod.Spec.NodeName, &index, ptpInterfacesOnly)
	}
	config.MaxL2GraphSize = index
	return nil
}

// Creates the Main topology graph
func (config *L2DiscoveryConfig) createL2InternalGraph(ptpInterfacesOnly bool) error {
	GlobalL2DiscoveryConfig.L2ConnectivityMap = graph.New(config.MaxL2GraphSize)
	for _, aPod := range config.L2DiscoveryPods {
		for iface, ifaceMap := range config.DiscoveryMap[aPod.Spec.NodeName][ExperimentalEthertype] {
			for mac := range ifaceMap.Remote {
				if v, ok := config.ClusterIndexToInt[l2.IfClusterIndex{InterfaceName: iface, NodeName: aPod.Spec.NodeName}]; ok {
					if w, ok := config.ClusterMacToInt[mac]; ok {
						if ptpInterfacesOnly &&
							(!config.PtpIfList[v].IfPTPCaps.HwRx ||
								!config.PtpIfList[v].IfPTPCaps.HwTx ||
								!config.PtpIfList[v].IfPTPCaps.HwRawClock ||
								!config.PtpIfList[w].IfPTPCaps.HwRx ||
								!config.PtpIfList[w].IfPTPCaps.HwTx ||
								!config.PtpIfList[w].IfPTPCaps.HwRawClock) {
							continue
						}
						config.L2ConnectivityMap.AddBoth(v, w)
					}
				}
			}
		}
	}
	// Init LANs
	out := graph.Components(config.L2ConnectivityMap)
	logrus.Infof("%v", out)
	config.LANs = &out
	config.PrintAllNICs()

	logrus.Infof("NIC num: %d", config.MaxL2GraphSize)
	return nil
}

// Gets the grandmaster port by using L2 discovery data for ptp ethertype
func (config *L2DiscoveryConfig) getInterfacesReceivingPTP(ptpInterfacesOnly bool) {
	for _, aPod := range config.L2DiscoveryPods {
		for _, ifaceMap := range config.DiscoveryMap[aPod.Spec.NodeName][PtpEthertype] {
			if len(ifaceMap.Remote) == 0 {
				continue
			}
			aPortGettingPTP := &l2.PtpIf{}
			aPortGettingPTP.Iface = ifaceMap.Local
			aPortGettingPTP.NodeName = aPod.Spec.NodeName
			aPortGettingPTP.InterfaceName = aPortGettingPTP.Iface.IfName

			if ptpInterfacesOnly &&
				(!aPortGettingPTP.IfPTPCaps.HwRx ||
					!aPortGettingPTP.IfPTPCaps.HwTx ||
					!aPortGettingPTP.IfPTPCaps.HwRawClock) {
				continue
			}
			config.PortsGettingPTP = append(config.PortsGettingPTP, aPortGettingPTP)
		}
	}
	logrus.Infof("interfaces receiving PTP frames: %v", config.PortsGettingPTP)
}

// Creates Mapping tables between interfaces index, mac address, and graph integer indexes
func (config *L2DiscoveryConfig) createMaps(disc map[string]map[string]*l2.Neighbors, nodeName string, index *int, ptpInterfacesOnly bool) {
	config.updateMaps(disc, nodeName, index, ExperimentalEthertype, ptpInterfacesOnly)
	config.updateMaps(disc, nodeName, index, LocalInterfaces, ptpInterfacesOnly)
	config.getInterfacesReceivingPTP(ptpInterfacesOnly)
}

// updates Mapping tables between interfaces index, mac address, and graph integer indexes for a given ethertype
func (config *L2DiscoveryConfig) updateMaps(disc map[string]map[string]*l2.Neighbors, nodeName string, index *int, ethertype string, ptpInterfacesOnly bool) {
	for _, ifaceData := range disc[ethertype] {
		if _, ok := config.ClusterMacToInt[ifaceData.Local.IfMac.Data]; ok {
			continue
		}

		aInterface := l2.PtpIf{}
		aInterface.NodeName = nodeName
		aInterface.InterfaceName = ifaceData.Local.IfName
		aInterface.Iface = ifaceData.Local

		if ptpInterfacesOnly &&
			(!aInterface.IfPTPCaps.HwRx ||
				!aInterface.IfPTPCaps.HwTx ||
				!aInterface.IfPTPCaps.HwRawClock) {
			continue
		}
		// create maps
		config.ClusterMacToInt[ifaceData.Local.IfMac.Data] = *index
		config.ClusterIndexToInt[l2.IfClusterIndex{InterfaceName: ifaceData.Local.IfName, NodeName: nodeName}] = *index
		config.ClusterMacs[l2.IfClusterIndex{InterfaceName: ifaceData.Local.IfName, NodeName: nodeName}] = ifaceData.Local.IfMac.Data
		config.ClusterIndexes[ifaceData.Local.IfMac.Data] = l2.IfClusterIndex{InterfaceName: ifaceData.Local.IfName, NodeName: nodeName}

		config.PtpIfList = append(config.PtpIfList, &aInterface)
		(*index)++
	}
}

// Gets the list of l2discovery pods
func (config *L2DiscoveryConfig) getL2TopologyDiscoveryPods() error {
	aPodList, err := l2client.Client.K8sClient.CoreV1().Pods(L2DiscoveryNsName).List(context.Background(), metav1.ListOptions{LabelSelector: "name=l2discovery"})
	if err != nil {
		return fmt.Errorf("could not get list of linkloop pods, err=%s", err)
	}
	for index := range aPodList.Items {
		config.L2DiscoveryPods[aPodList.Items[index].Spec.NodeName] = &aPodList.Items[index]
	}
	return nil
}
