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
	definition netattdefv1.NetworkAttachmentDefinition
	configs    []string
}

func NewNetworkAttachmentDefinitionBuilder(namespace, nadName string) *NetworkAttachmentDefinitionBuilder {
	return &NetworkAttachmentDefinitionBuilder{
		configs: []string{},
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

func (b *NetworkAttachmentDefinitionBuilder) WithAdditonalPlugin(config string) *NetworkAttachmentDefinitionBuilder {
	b.configs = append(b.configs, config)
	return b
}

func (b *NetworkAttachmentDefinitionBuilder) WithTuning(sysctls string) *NetworkAttachmentDefinitionBuilder {
	b.configs = append(b.configs, sysctls)
	return b
}

func (b *NetworkAttachmentDefinitionBuilder) WithBond(bondName, link1, link2 string) *NetworkAttachmentDefinitionBuilder {
	b.configs = append(b.configs, fmt.Sprintf(`{
	    "type": "bond",
		"ifname": "%s",
		"mode": "active-backup",
		"failOverMac": 1,
		"linksInContainer": true,
		"miimon": "100",
		"links": [ {"name": "%s"}, {"name": "%s"} ],
        "ipam": {
            "type": "host-local",
            "subnet": "1.1.1.0/24"
        }}`, bondName, link1, link2))
	return b
}

func (b *NetworkAttachmentDefinitionBuilder) WithMacVlan(ip string) *NetworkAttachmentDefinitionBuilder {
	b.configs = append(b.configs, fmt.Sprintf(`{"type": "macvlan","ipam":{"type":"static","addresses":[{"address":"%s/24"}]}}`, ip))
	return b
}

func (b *NetworkAttachmentDefinitionBuilder) Build() (*netattdefv1.NetworkAttachmentDefinition, error) {
	if len(b.configs) == 0 {
		return nil, errors.New("NetworkAttachmentDefinition with no configs")
	}

	b.definition.Spec.Config = fmt.Sprintf(`{"cniVersion":"0.4.0","name":"%s","plugins":[%s]}`, b.definition.ObjectMeta.Name, strings.Join(b.configs, ", "))
	return &b.definition, nil
}

func SysctlConfig(sysctls map[string]string) (string, error) {
	sysctlString, err := json.Marshal(sysctls)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`{"type":"tuning","sysctl":%s}`, string(sysctlString)), nil
}
