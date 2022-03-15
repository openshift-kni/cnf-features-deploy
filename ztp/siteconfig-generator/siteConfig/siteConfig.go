package siteConfig

import (
	"fmt"
	"reflect"
	"strings"
)

const localExtraManifestPath = "extra-manifest"
const workloadPath = "workload"
const workloadFile = "03-workload-partitioning.yaml"
const workloadCrioFile = "crio.conf"
const workloadKubeletFile = "kubelet.conf"
const cpuset = "$cpuset"
const SNO = "sno"
const Standard = "standard"
const Master = "master"

var Separator = []byte("---\n")

func (sc *SiteConfig) GetSiteConfigFieldValue(path string, clusterId int, nodeId int) (interface{}, error) {
	keys := strings.Split(path, ".")
	v := reflect.ValueOf(sc)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for _, key := range keys[1:] {
		if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
			intf := v.Interface()

			arrClusters, ok := intf.([]Clusters)
			if ok {
				if clusterId < 0 || clusterId >= len(arrClusters) {
					return nil, fmt.Errorf("Cluster ID out of range: %d", clusterId)
				}
				v = reflect.ValueOf(arrClusters[clusterId])
			}

			arrNodes, ok := intf.([]Nodes)
			if ok {
				if nodeId < 0 || nodeId >= len(arrNodes) {
					return nil, fmt.Errorf("Node ID out of range: %d", nodeId)
				}
				v = reflect.ValueOf(arrNodes[nodeId])
			}

			v = v.FieldByName(key)
		} else if v.Kind() == reflect.Struct {
			v = v.FieldByName(key)
		} else if v.Kind() != reflect.Struct {
			return nil, fmt.Errorf("only accepts structs; got %T", v)
		}
	}

	if !v.IsValid() || v.IsZero() {
		return "", nil
	}

	return v.Interface(), nil
}

// SiteConfig
type SiteConfig struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

func (sc *SiteConfig) areAllOverridesValid(validKinds, validNodeKinds *map[string]bool) error {
	err := areOverridesValid(&sc.Spec.CrTemplates, validKinds)
	if err != nil {
		return fmt.Errorf("Invalid override in SiteConfig.Spec: %w", err)
	}
	for clusterIndex, cluster := range sc.Spec.Clusters {
		err = areOverridesValid(&cluster.CrTemplates, validKinds)
		if err != nil {
			return fmt.Errorf("Invalid override in SiteConfig.Spec.Clusters[%d]: %w", clusterIndex, err)
		}
		for nodeIndex, node := range cluster.Nodes {
			err = areOverridesValid(&node.CrTemplates, validNodeKinds)
			if err != nil {
				return fmt.Errorf("Invalid override in SiteConfig.Spec.Clusters[%d].Nodes[%d]: %w", clusterIndex, nodeIndex, err)
			}
		}
	}
	return nil
}

func areOverridesValid(overrides *map[string]string, validKinds *map[string]bool) error {
	for override := range *overrides {
		if _, ok := (*validKinds)[override]; !ok {
			return fmt.Errorf("%q is not a valid CR type to override", override)
		}
	}
	return nil
}

// Metadata
type Metadata struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Labels    map[string]string `yaml:"labels"`
}

// Spec
type Spec struct {
	PullSecretRef          PullSecretRef          `yaml:"pullSecretRef"`
	ClusterImageSetNameRef string                 `yaml:"clusterImageSetNameRef"`
	SshPublicKey           string                 `yaml:"sshPublicKey"`
	SshPrivateKeySecretRef SshPrivateKeySecretRef `yaml:"sshPrivateKeySecretRef"`
	Clusters               []Clusters             `yaml:"clusters"`
	BaseDomain             string                 `yaml:"baseDomain"`
	CrTemplates            map[string]string      `yaml:"crTemplates"`
	BiosConfigRef          BiosConfigRef          `yaml:"biosConfigRef"`
}

// Lookup a specific CR template for this site
func (site *Spec) CrTemplateSearch(kind string) (string, bool) {
	template, ok := site.CrTemplates[kind]
	return template, ok
}

// Lookup bios config file path for this site
func (site *Spec) BiosFileSearch() string {
	return site.BiosConfigRef.FilePath
}

// PullSecretRef
type PullSecretRef struct {
	Name string `yaml:"name"`
}

// SshPrivateKeySecretRef
type SshPrivateKeySecretRef struct {
	Name string `yaml:"name"`
}

// Clusters
type Clusters struct {
	ApiVIP                 string            `yaml:"apiVIP"`
	IngressVIP             string            `yaml:"ingressVIP"`
	ClusterName            string            `yaml:"clusterName"`
	AdditionalNTPSources   []string          `yaml:"additionalNTPSources"`
	Nodes                  []Nodes           `yaml:"nodes"`
	MachineNetwork         []MachineNetwork  `yaml:"machineNetwork"`
	ServiceNetwork         []string          `yaml:"serviceNetwork"`
	ClusterLabels          map[string]string `yaml:"clusterLabels"`
	NetworkType            string            `yaml:"networkType"`
	ClusterNetwork         []ClusterNetwork  `yaml:"clusterNetwork"`
	IgnitionConfigOverride string            `yaml:"ignitionConfigOverride"`
	DiskEncryption         DiskEncryption    `yaml:"diskEncryption"`
	ProxySettings          ProxySettings     `yaml:"proxy,omitempty"`
	ExtraManifestPath      string            `yaml:"extraManifestPath"`
	ClusterImageSetNameRef string            `yaml:"clusterImageSetNameRef,omitempty"`
	BiosConfigRef          BiosConfigRef     `yaml:"biosConfigRef"`

	NumMasters  uint8
	NumWorkers  uint8
	ClusterType string
	CrTemplates map[string]string `yaml:"crTemplates"`
}

// Provide custom YAML unmarshal for Clusters which provides default values
func (rv *Clusters) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type ClusterDefaulted Clusters
	var defaults = ClusterDefaulted{
		NetworkType: "OVNKubernetes",
	}

	out := defaults
	err := unmarshal(&out)
	if err != nil {
		return err
	}
	*rv = Clusters(out)
	// Tally master and worker counts based on node roles
	rv.NumMasters = 0
	rv.NumWorkers = 0
	for _, node := range rv.Nodes {
		if len(node.Role) == 0 || node.Role == Master {
			// The default role (if it's not set) is master
			rv.NumMasters += 1
		} else {
			rv.NumWorkers += 1
		}
	}
	if rv.NumMasters != 1 && rv.NumMasters != 3 {
		return fmt.Errorf("Number of masters (counted %d) must be exactly 1 or 3", rv.NumMasters)
	}
	// Autodetect ClusterType based on the node counts
	if rv.NumMasters == 1 && rv.NumWorkers == 0 {
		rv.ClusterType = SNO
	} else {
		rv.ClusterType = Standard
	}
	return nil
}

// Lookup a specific CR template for this cluster, with fallback to site
func (cluster *Clusters) CrTemplateSearch(kind string, site *Spec) (string, bool) {
	template, ok := cluster.CrTemplates[kind]
	if ok {
		return template, ok
	}
	return site.CrTemplateSearch(kind)
}

// Lookup bios config file path for this cluster, with fallback to site
func (cluster *Clusters) BiosFileSearch(site *Spec) string {
	filepath := cluster.BiosConfigRef.FilePath
	if filepath != "" {
		return filepath
	}
	return site.BiosFileSearch()
}

type DiskEncryption struct {
	Type string       `yaml:"type"`
	Tang []TangConfig `yaml:"tang"`
}

// Provide custom YAML unmarshal for DiskEncryption which provides default values
func (rv *DiskEncryption) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type ValueDefaulted DiskEncryption
	var defaults = ValueDefaulted{
		Type: "none",
	}

	out := defaults
	err := unmarshal(&out)
	*rv = DiskEncryption(out)
	return err
}

type ProxySettings struct {
	HttpProxy  string `yaml:"httpProxy,omitempty"`
	HttpsProxy string `yaml:"httpsProxy,omitempty"`
	NoProxy    string `yaml:"noProxy,omitempty"`
}

type TangConfig struct {
	URL        string `yaml:"url" json:"url"`
	Thumbprint string `yaml:"thumbprint" json:"thp"`
}

// Nodes
type Nodes struct {
	BmcAddress             string                 `yaml:"bmcAddress"`
	BootMACAddress         string                 `yaml:"bootMACAddress"`
	RootDeviceHints        map[string]interface{} `yaml:"rootDeviceHints"`
	Cpuset                 string                 `yaml:"cpuset"`
	NodeNetwork            NodeNetwork            `yaml:"nodeNetwork"`
	HostName               string                 `yaml:"hostName"`
	BmcCredentialsName     BmcCredentialsName     `yaml:"bmcCredentialsName"`
	BootMode               string                 `yaml:"bootMode"`
	UserData               map[string]interface{} `yaml:"userData"`
	InstallerArgs          string                 `yaml:"installerArgs"`
	IgnitionConfigOverride string                 `yaml:"ignitionConfigOverride"`
	Role                   string                 `yaml:"role"`
	CrTemplates            map[string]string      `yaml:"crTemplates"`
	BiosConfigRef          BiosConfigRef          `yaml:"biosConfigRef"`
}

// Provide custom YAML unmarshal for Nodes which provides default values
func (rv *Nodes) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type ValueDefaulted Nodes
	var defaults = ValueDefaulted{
		BootMode: "UEFI",
		Role:     "master",
	}

	out := defaults
	err := unmarshal(&out)
	*rv = Nodes(out)
	return err
}

// Lookup a specific CR template for this node, with fallback to cluster and site
func (node *Nodes) CrTemplateSearch(kind string, cluster *Clusters, site *Spec) (string, bool) {
	template, ok := node.CrTemplates[kind]
	if ok {
		return template, ok
	}
	return cluster.CrTemplateSearch(kind, site)
}

// Return true if the NodeNetwork content is empty or not defined
func (node *Nodes) nodeNetworkIsEmpty() bool {
	if len(node.NodeNetwork.Config) == 0 && len(node.NodeNetwork.Interfaces) == 0 {
		return true
	}
	return false
}

// Lookup bios config file path for this node, with fallback to cluster and site
func (node *Nodes) BiosFileSearch(cluster *Clusters, site *Spec) string {
	filepath := node.BiosConfigRef.FilePath
	if filepath != "" {
		return filepath
	}
	return cluster.BiosFileSearch(site)
}

// MachineNetwork
type MachineNetwork struct {
	Cidr string `yaml:"cidr"`
}

// ClusterNetwork
type ClusterNetwork struct {
	Cidr       string `yaml:"cidr"`
	HostPrefix int    `yaml:"hostPrefix"`
}

// NodeNetwork
type NodeNetwork struct {
	Config     map[string]interface{} `yaml:"config"`
	Interfaces []Interfaces           `yaml:"interfaces"`
}

// BmcCredentialsName
type BmcCredentialsName struct {
	Name string `yaml:"name"`
}

// Interfaces
type Interfaces struct {
	Name       string `yaml:"name"`
	MacAddress string `yaml:"macAddress"`
}

// BiosConfigRef
type BiosConfigRef struct {
	FilePath string `yaml:"filePath"`
}
