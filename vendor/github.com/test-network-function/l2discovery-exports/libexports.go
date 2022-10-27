package libexports

import (
	"fmt"
	"strings"
)

type Mac struct {
	Data string
}

type PCIAddress struct {
	Device, Function, Description string
}

type PTPCaps struct {
	HwRx, HwTx, HwRawClock bool
}

type Iface struct {
	IfName    string
	IfMac     Mac
	IfIndex   int
	IfPci     PCIAddress
	IfPTPCaps PTPCaps
	IfUp      bool
}

type Neighbors struct {
	Local  Iface
	Remote map[string]bool
}

func (mac Mac) String() string {
	return strings.ToUpper(string([]byte(mac.Data)[0:2]) + ":" +
		string([]byte(mac.Data)[2:4]) + ":" +
		string([]byte(mac.Data)[4:6]) + ":" +
		string([]byte(mac.Data)[6:8]) + ":" +
		string([]byte(mac.Data)[8:10]) + ":" +
		string([]byte(mac.Data)[10:12]))
}

// Object representing a ptp interface within a cluster.
type PtpIf struct {
	// Index of the interface in the cluster (node/interface name)
	IfClusterIndex
	// Interface
	Iface
}

// Object used to index interfaces in a cluster
type IfClusterIndex struct {
	// interface name
	InterfaceName string
	// node name
	NodeName string
}

func (index IfClusterIndex) String() string {
	return fmt.Sprintf("%s_%s", index.NodeName, index.InterfaceName)
}

func (iface *PtpIf) String() string {
	return fmt.Sprintf("%s : %s", iface.NodeName, iface.IfName)
}

func (iface *PtpIf) String1() string {
	return fmt.Sprintf("index:%s mac:%s", iface.IfClusterIndex, iface.IfMac)
}
