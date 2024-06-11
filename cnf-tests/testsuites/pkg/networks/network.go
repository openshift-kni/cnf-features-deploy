package networks

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	netattdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NetworkAttachmentDefinitionBuilder struct {
	definition        netattdefv1.NetworkAttachmentDefinition
	config            string
	metaPluginConfigs []string
	ipam              string
	errorMsg          string
}

func NewNetworkAttachmentDefinitionBuilder(namespace, nadName string) *NetworkAttachmentDefinitionBuilder {
	return &NetworkAttachmentDefinitionBuilder{
		metaPluginConfigs: []string{},
		definition: netattdefv1.NetworkAttachmentDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nadName,
				Namespace: namespace,
			},
			Spec: netattdefv1.NetworkAttachmentDefinitionSpec{
				Config: "",
			},
		},
	}
}

func (b *NetworkAttachmentDefinitionBuilder) WithTuning(sysctls string) *NetworkAttachmentDefinitionBuilder {
	b.metaPluginConfigs = append(b.metaPluginConfigs, fmt.Sprintf("{%s}", sysctls))
	return b
}

func (b *NetworkAttachmentDefinitionBuilder) WithBond(bondName, link1, link2 string, mtu int) *NetworkAttachmentDefinitionBuilder {
	bondConfig := `
	    "type": "bond",
		"ifname": "%s",
		"mode": "active-backup",
		"failOverMac": 1,
		"linksInContainer": true,
		"miimon": "100",
		"mtu": %d,
		"links": [ {"name": "%s"}, {"name": "%s"} ]`
	b.setConfig(fmt.Sprintf(bondConfig, bondName, mtu, link1, link2))
	return b
}

func (b *NetworkAttachmentDefinitionBuilder) WithStaticIpam(ip string) *NetworkAttachmentDefinitionBuilder {
	b.ipam = fmt.Sprintf(`"ipam": {"type":"static","addresses":[{"address":"%s/24"}]},`, ip)
	return b
}

func (b *NetworkAttachmentDefinitionBuilder) WithHostLocalIpam(ip string) *NetworkAttachmentDefinitionBuilder {
	b.ipam = fmt.Sprintf(`"ipam": {"type": "host-local", "subnet": "%s/24"},`, ip)
	return b
}

func (b *NetworkAttachmentDefinitionBuilder) WithMacVlan() *NetworkAttachmentDefinitionBuilder {
	b.setConfig(`"type": "macvlan"`)
	return b
}

func (b *NetworkAttachmentDefinitionBuilder) WithVlan(master string, vlanID int, linkInContainer bool) *NetworkAttachmentDefinitionBuilder {
	b.setConfig(fmt.Sprintf(`"type": "vlan", "master": "%s", "vlanId": %d, "linkInContainer": %t`, master, vlanID, linkInContainer))
	return b
}

func (b *NetworkAttachmentDefinitionBuilder) WithTap() *NetworkAttachmentDefinitionBuilder {
	b.setConfig(`"type": "tap", "selinuxcontext": "system_u:system_r:container_t:s0", "multiQueue": true`)
	return b
}

func (b *NetworkAttachmentDefinitionBuilder) Build() (*netattdefv1.NetworkAttachmentDefinition, error) {
	if b.errorMsg != "" {
		return nil, errors.New(b.errorMsg)
	}
	configs := []string{fmt.Sprintf("{%s %s}", b.ipam, b.config)}
	configs = append(configs, b.metaPluginConfigs...)
	b.definition.Spec.Config = fmt.Sprintf(`{"cniVersion":"0.4.0","name":"%s","plugins":[%s]}`, b.definition.ObjectMeta.Name, strings.Join(configs, ", "))
	return &b.definition, nil
}

func SysctlConfig(sysctls map[string]string) (string, error) {
	sysctlString, err := json.Marshal(sysctls)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`"type":"tuning","sysctl":%s`, string(sysctlString)), nil
}

func (b *NetworkAttachmentDefinitionBuilder) setConfig(config string) {
	if b.config != "" {
		b.errorMsg = "Main plugin set more than twice"
	}
	b.config = config
}
