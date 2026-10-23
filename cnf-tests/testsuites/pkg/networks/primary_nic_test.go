package networks

import (
	"testing"

	sriovv1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestIsPrimaryNICName(t *testing.T) {
	tests := []struct {
		name    string
		device  string
		primary []string
		want    bool
	}{
		{name: "exact match", device: "ens3f0", primary: []string{"ens3f0"}, want: true},
		{name: "netdevport alias", device: "ens2f0np0", primary: []string{"ens2f0"}, want: true},
		{name: "primary is netdevport", device: "ens7f0", primary: []string{"ens7f0np0"}, want: true},
		{name: "same card needs PCI not name", device: "ens8f1", primary: []string{"ens8f0"}, want: false},
		{name: "different ens card", device: "ens1f0", primary: []string{"ens8f0"}, want: false},
		{name: "eth exact only", device: "eth0", primary: []string{"eth0"}, want: true},
		{name: "eth sibling by name", device: "eth1", primary: []string{"eth0"}, want: false},
		{name: "empty primary", device: "ens3f0", primary: []string{""}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isPrimaryNICName(tt.device, tt.primary...))
		})
	}
}

func TestSplitPrimaryNICNames(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "newline sorted unique", in: "ens2f0\nens1f0\n", want: []string{"ens1f0", "ens2f0"}},
		{name: "crlf", in: "ens7f0\r\n", want: []string{"ens7f0"}},
		{name: "spaces dedupe", in: "ens1f0 ens2f0 ens1f0", want: []string{"ens1f0", "ens2f0"}},
		{name: "blank", in: " \n\t", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, splitPrimaryNICNames(tt.in))
		})
	}
}

func TestPciCardID(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "function 0", in: "0000:86:00.0", want: "0000:86:00"},
		{name: "function 3", in: "0000:86:00.3", want: "0000:86:00"},
		{name: "empty", in: "", want: ""},
		{name: "invalid", in: "invalid", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, pciCardID(tt.in))
		})
	}
}

func TestExcludePrimarySriovDevices(t *testing.T) {
	tests := []struct {
		name     string
		devices  []*sriovv1.InterfaceExt
		primary  []string
		wantKeep []string
	}{
		{
			name: "ens card siblings by PCI",
			devices: []*sriovv1.InterfaceExt{
				{Name: "ens1f0", PciAddress: "0000:3b:00.0"},
				{Name: "ens1f1", PciAddress: "0000:3b:00.1"},
				{Name: "ens3f0", PciAddress: "0000:86:00.0"},
				{Name: "ens3f1", PciAddress: "0000:86:00.1"},
				{Name: "ens3f2", PciAddress: "0000:86:00.2"},
				{Name: "ens8f0np0", PciAddress: "0000:d8:00.0"},
			},
			primary:  []string{"ens3f0"},
			wantKeep: []string{"ens1f0", "ens1f1", "ens8f0np0"},
		},
		{
			name: "bonded br-ex two cards",
			devices: []*sriovv1.InterfaceExt{
				{Name: "ens1f0", PciAddress: "0000:3b:00.0"},
				{Name: "ens1f1", PciAddress: "0000:3b:00.1"},
				{Name: "ens2f0", PciAddress: "0000:3c:00.0"},
				{Name: "ens2f1", PciAddress: "0000:3c:00.1"},
				{Name: "ens7f0", PciAddress: "0000:af:00.0"},
			},
			primary:  []string{"ens1f0", "ens2f0"},
			wantKeep: []string{"ens7f0"},
		},
		{
			name: "netdevport primary maps to PCI card",
			devices: []*sriovv1.InterfaceExt{
				{Name: "ens8f0np0", PciAddress: "0000:d8:00.0"},
				{Name: "ens8f1np1", PciAddress: "0000:d8:00.1"},
				{Name: "ens1f0", PciAddress: "0000:3b:00.0"},
			},
			primary:  []string{"ens8f0"},
			wantKeep: []string{"ens1f0"},
		},
		{
			name: "unexpanded bond name excludes nothing",
			devices: []*sriovv1.InterfaceExt{
				{Name: "ens1f0", PciAddress: "0000:3b:00.0"},
				{Name: "ens8f0", PciAddress: "0000:d8:00.0"},
			},
			primary:  []string{"bond0"},
			wantKeep: []string{"ens1f0", "ens8f0"},
		},
		{
			name: "eth siblings by PCI card",
			devices: []*sriovv1.InterfaceExt{
				{Name: "eth0", PciAddress: "0000:03:00.0"},
				{Name: "eth1", PciAddress: "0000:03:00.1"},
				{Name: "ens1f0", PciAddress: "0000:3b:00.0"},
			},
			primary:  []string{"eth0"},
			wantKeep: []string{"ens1f0"},
		},
		{
			name: "different PCI slots kept",
			devices: []*sriovv1.InterfaceExt{
				{Name: "eth0", PciAddress: "0000:03:00.0"},
				{Name: "eth1", PciAddress: "0000:03:01.0"},
			},
			primary:  []string{"eth0"},
			wantKeep: []string{"eth1"},
		},
		{
			name: "without PciAddress only exact name drops",
			devices: []*sriovv1.InterfaceExt{
				{Name: "ens3f0"},
				{Name: "ens3f1"},
				{Name: "ens1f0"},
			},
			primary:  []string{"ens3f0"},
			wantKeep: []string{"ens3f1", "ens1f0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deviceNames(ExcludePrimarySriovDevices(tt.devices, tt.primary))
			assert.Equal(t, tt.wantKeep, got)
		})
	}
}

func deviceNames(devices []*sriovv1.InterfaceExt) []string {
	names := make([]string, 0, len(devices))
	for _, d := range devices {
		names = append(names, d.Name)
	}
	return names
}
