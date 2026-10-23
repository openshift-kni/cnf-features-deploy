package networks

import (
	"fmt"
	"sort"
	"strings"

	sriovv1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/nodes"
	corev1 "k8s.io/api/core/v1"
)

// brExPrimaryNICsScript lists br-ex PFs (expands OVS/kernel bonds).
const brExPrimaryNICsScript = `
set -euo pipefail
for p in $(chroot /host /usr/bin/ovs-vsctl list-ports br-ex | /usr/bin/grep -v patch); do
  uuids=$(chroot /host /usr/bin/ovs-vsctl get port "$p" interfaces | tr -d '[]',)
  for uuid in $uuids; do
    iface=$(chroot /host /usr/bin/ovs-vsctl get interface "$uuid" name | tr -d '"')
    if [ -r "/host/sys/class/net/${iface}/bonding/slaves" ]; then
      cat "/host/sys/class/net/${iface}/bonding/slaves"
      printf '\n'
    else
      printf '%s\n' "$iface"
    fi
  done
done | tr ' ' '\n' | sed '/^$/d' | sort -u
`

// GetPrimaryNICsForNode returns br-ex PFs via the sriov-config-daemon (bonds expanded).
func GetPrimaryNICsForNode(node *corev1.Node) ([]string, error) {
	buf, err := nodes.ExecCommandOnNodeViaSriovDaemon(client.Client, node,
		[]string{"/bin/bash", "-c", brExPrimaryNICsScript})
	if err != nil {
		return nil, err
	}

	nics := splitPrimaryNICNames(string(buf))
	if len(nics) == 0 {
		return nil, fmt.Errorf("no physical NIC found for br-ex on node %s", node.Name)
	}

	return nics, nil
}

// ExcludePrimarySriovDevices drops br-ex PFs and same-card siblings from devices.
// Same-card siblings are detected via PciAddress (domain:bus:device); names are
// matched only for the primary itself and netdevport aliases (...npN).
func ExcludePrimarySriovDevices(devices []*sriovv1.InterfaceExt, primaryNICs []string) []*sriovv1.InterfaceExt {
	primaryCards := primaryPCICardIDs(devices, primaryNICs)

	filtered := make([]*sriovv1.InterfaceExt, 0, len(devices))
	for _, device := range devices {
		if isPrimaryNICName(device.Name, primaryNICs...) {
			continue
		}
		if id := pciCardID(device.PciAddress); id != "" {
			if _, ok := primaryCards[id]; ok {
				continue
			}
		}
		filtered = append(filtered, device)
	}

	return filtered
}

// primaryPCICardIDs returns PCI card IDs for devices whose name matches a br-ex primary.
func primaryPCICardIDs(devices []*sriovv1.InterfaceExt, primaryNICs []string) map[string]struct{} {
	cards := make(map[string]struct{})
	for _, device := range devices {
		if !isPrimaryNICName(device.Name, primaryNICs...) {
			continue
		}
		if id := pciCardID(device.PciAddress); id != "" {
			cards[id] = struct{}{}
		}
	}

	return cards
}

// pciCardID returns domain:bus:device for a PCI address (function stripped).
// Example: 0000:86:00.1 -> 0000:86:00
func pciCardID(pciAddress string) string {
	pciAddress = strings.TrimSpace(pciAddress)
	if pciAddress == "" {
		return ""
	}
	i := strings.LastIndex(pciAddress, ".")
	if i <= 0 {
		return ""
	}

	return pciAddress[:i]
}

// isPrimaryNICName is true if deviceName is a br-ex PF name or its netdevport alias.
func isPrimaryNICName(deviceName string, primaryNICs ...string) bool {
	for _, primaryNIC := range primaryNICs {
		if primaryNIC == "" {
			continue
		}
		if deviceName == primaryNIC {
			return true
		}
		// ens5f0 <-> ens5f0np0 (do not treat "np" inside enp* as netdevport).
		if strings.HasPrefix(deviceName, primaryNIC+"np") || strings.HasPrefix(primaryNIC, deviceName+"np") {
			return true
		}
	}

	return false
}

func splitPrimaryNICNames(out string) []string {
	fields := strings.Fields(strings.ReplaceAll(out, "\r", ""))
	if len(fields) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(fields))
	nics := make([]string, 0, len(fields))
	for _, name := range fields {
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		nics = append(nics, name)
	}
	sort.Strings(nics)

	return nics
}
