package main

import (
	"flag"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SiteConfig represents the structure of a SiteConfig CRD
type SiteConfig struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

// Metadata represents the metadata section of a SiteConfig
type Metadata struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Labels    map[string]string `yaml:"labels"`
}

// Spec represents the spec section of a SiteConfig
type Spec struct {
	PullSecretRef          PullSecretRef          `yaml:"pullSecretRef"`
	ClusterImageSetNameRef string                 `yaml:"clusterImageSetNameRef"`
	SshPublicKey           string                 `yaml:"sshPublicKey"`
	SshPrivateKeySecretRef SshPrivateKeySecretRef `yaml:"sshPrivateKeySecretRef"`
	Clusters               []Cluster              `yaml:"clusters"`
	BaseDomain             string                 `yaml:"baseDomain"`
	CrTemplates            map[string]string      `yaml:"crTemplates"`
	CrAnnotations          CrAnnotations          `yaml:"crAnnotations"`
	BiosConfigRef          BiosConfigRef          `yaml:"biosConfigRef"`
}

// PullSecretRef represents the pull secret reference
type PullSecretRef struct {
	Name string `yaml:"name"`
}

// SshPrivateKeySecretRef represents the SSH private key secret reference
type SshPrivateKeySecretRef struct {
	Name string `yaml:"name"`
}

// CrAnnotations represents custom resource annotations
type CrAnnotations struct {
	Add map[string]map[string]string `yaml:"add"`
}

// BiosConfigRef represents BIOS configuration reference
type BiosConfigRef struct {
	FilePath string `yaml:"filePath"`
}

// CPUPartitioningMode is used to drive how a cluster nodes CPUs are Partitioned.
type CPUPartitioningMode string

const (
	// The only supported configurations are an all or nothing configuration.
	CPUPartitioningNone     CPUPartitioningMode = "None"
	CPUPartitioningAllNodes CPUPartitioningMode = "AllNodes"
)

// ManifestsConfigMapReference is a reference to a manifests ConfigMap
type ManifestsConfigMapReference struct {
	// Name is the name of the ConfigMap that this refers to
	Name string `json:"name"`
}

// IronicInspect represents ironic inspect configuration
type IronicInspect string

const (
	InspectDisabled IronicInspect = "disabled"
	InspectEnabled  IronicInspect = "enabled"
)

// Filter represents extra manifests filter configuration
type Filter struct {
	InclusionDefault *string  `yaml:"inclusionDefault"`
	Exclude          []string `yaml:"exclude"`
	Include          []string `yaml:"include"`
}

// SiteConfigMap represents site config map configuration
type SiteConfigMap struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Data      map[string]string `yaml:"data"`
}

// Cluster represents a cluster configuration
type Cluster struct {
	ApiVIP                 string                `yaml:"apiVIP"`
	IngressVIP             string                `yaml:"ingressVIP"`
	ApiVIPs                []string              `yaml:"apiVIPs"`
	IngressVIPs            []string              `yaml:"ingressVIPs"`
	ClusterName            string                `yaml:"clusterName"`
	HoldInstallation       bool                  `yaml:"holdInstallation"`
	AdditionalNTPSources   []string              `yaml:"additionalNTPSources"`
	Nodes                  []Node                `yaml:"nodes"`
	MachineNetwork         []MachineNetworkEntry `yaml:"machineNetwork"`
	ServiceNetwork         []string              `yaml:"serviceNetwork"`
	ClusterLabels          map[string]string     `yaml:"clusterLabels"`
	NetworkType            string                `yaml:"networkType"`
	InstallConfigOverrides string                `yaml:"installConfigOverrides,omitempty"`
	ClusterNetwork         []ClusterNetwork      `yaml:"clusterNetwork"`
	IgnitionConfigOverride string                `yaml:"ignitionConfigOverride"`
	DiskEncryption         DiskEncryption        `yaml:"diskEncryption"`
	Proxy                  Proxy                 `yaml:"proxy,omitempty"`
	ExtraManifestPath      string                `yaml:"extraManifestPath"`
	ClusterImageSetNameRef string                `yaml:"clusterImageSetNameRef,omitempty"`
	BiosConfigRef          BiosConfigRef         `yaml:"biosConfigRef"`
	ExtraManifests         ExtraManifests        `yaml:"extraManifests"`
	CPUPartitioningMode    CPUPartitioningMode   `yaml:"cpuPartitioningMode"`
	SiteConfigMap          SiteConfigMap         `yaml:"siteConfigMap"`
	PlatformType           string                `yaml:"platformType,omitempty"`
	CPUArchitecture        string                `yaml:"cpuArchitecture,omitempty"`

	ExtraManifestOnly          bool                          `yaml:"extraManifestOnly,omitempty"`
	NumMasters                 uint8                         `yaml:"numMasters,omitempty"`
	NumWorkers                 uint8                         `yaml:"numWorkers,omitempty"`
	ClusterType                string                        `yaml:"clusterType,omitempty"`
	CrTemplates                map[string]string             `yaml:"crTemplates,omitempty"`
	CrAnnotations              CrAnnotations                 `yaml:"crAnnotations,omitempty"`
	CrSuppression              []string                      `yaml:"crSuppression,omitempty"`
	ManifestsConfigMapRefs     []ManifestsConfigMapReference `yaml:"manifestsConfigMapRefs,omitempty"`
	MergeDefaultMachineConfigs bool                          `yaml:"mergeDefaultMachineConfigs,omitempty"`
}

// MachineNetworkEntry represents a machine network entry
type MachineNetworkEntry struct {
	CIDR string `yaml:"cidr"`
}

// ClusterNetwork represents cluster network configuration
type ClusterNetwork struct {
	CIDR       string `yaml:"cidr"`
	HostPrefix int    `yaml:"hostPrefix"`
}

// DiskEncryption represents disk encryption configuration
type DiskEncryption struct {
	Type string       `yaml:"type"`
	Tang []TangServer `yaml:"tang"`
	Tpm2 TPM2Config   `yaml:"tpm2"`
}

// TangServer represents a Tang server configuration
type TangServer struct {
	URL        string `yaml:"url"`
	Thumbprint string `yaml:"thumbprint"`
}

// TPM2Config represents TPM2 configuration for disk encryption
type TPM2Config struct {
	PCRList string `yaml:"pcrList" json:"pcrList"`
}

// Proxy represents proxy configuration
type Proxy struct {
	HTTPProxy  string `yaml:"httpProxy"`
	HTTPSProxy string `yaml:"httpsProxy"`
	NoProxy    string `yaml:"noProxy"`
}

// ExtraManifests represents extra manifests configuration
type ExtraManifests struct {
	SearchPaths *[]string `yaml:"searchPaths"`
	Filter      *Filter   `yaml:"filter"`
}

// Node represents a node configuration
type Node struct {
	BmcAddress             string                 `yaml:"bmcAddress"`
	BootMACAddress         string                 `yaml:"bootMACAddress"`
	AutomatedCleaningMode  string                 `yaml:"automatedCleaningMode"`
	RootDeviceHints        map[string]interface{} `yaml:"rootDeviceHints"`
	Cpuset                 string                 `yaml:"cpuset"`
	NodeNetwork            NodeNetwork            `yaml:"nodeNetwork"`
	NodeLabels             map[string]string      `yaml:"nodeLabels"`
	HostName               string                 `yaml:"hostName"`
	BmcCredentialsName     BmcCredentialsName     `yaml:"bmcCredentialsName"`
	BootMode               string                 `yaml:"bootMode"`
	UserData               map[string]interface{} `yaml:"userData"`
	InstallerArgs          string                 `yaml:"installerArgs"`
	IgnitionConfigOverride string                 `yaml:"ignitionConfigOverride"`
	Role                   string                 `yaml:"role"`
	CrTemplates            map[string]string      `yaml:"crTemplates"`
	CrAnnotations          CrAnnotations          `yaml:"crAnnotations"`
	CrSuppression          []string               `yaml:"crSuppression"`
	BiosConfigRef          BiosConfigRef          `yaml:"biosConfigRef"`
	DiskPartition          []DiskPartition        `yaml:"diskPartition"`
	IronicInspect          IronicInspect          `yaml:"ironicInspect"`
}

// NodeNetwork represents node network configuration
type NodeNetwork struct {
	Config     map[string]interface{} `yaml:"config"`
	Interfaces []NetworkInterface     `yaml:"interfaces"`
}

// NetworkInterface represents a network interface
type NetworkInterface struct {
	Name       string `yaml:"name"`
	MacAddress string `yaml:"macAddress"`
}

// BmcCredentialsName represents BMC credentials name
type BmcCredentialsName struct {
	Name string `yaml:"name"`
}

// DiskPartition represents disk partition configuration
type DiskPartition struct {
	Device     string      `yaml:"device"`
	Partitions []Partition `yaml:"partitions"`
}

// Partition represents a partition configuration
type Partition struct {
	MountPoint       string `yaml:"mount_point"`
	Size             int    `yaml:"size"`
	Start            int    `yaml:"start"`
	FileSystemFormat string `yaml:"file_system_format"`
	MountFileName    string `yaml:"-"`
	Label            string `yaml:"-"`
	Encryption       bool   `yaml:"-"` // TODO: a place holder to enable disk encryption
}

func main() {
	var (
		outputDir           = flag.String("d", ".", "Output directory for converted ClusterInstance files")
		clusterTemplate     = flag.String("t", "open-cluster-management/ai-cluster-templates-v1", "Comma-separated list of template references for Cluster (format: namespace/name,namespace/name,...)")
		nodeTemplate        = flag.String("n", "open-cluster-management/ai-node-templates-v1", "Comma-separated list of template references for Nodes (format: namespace/name,namespace/name,...)")
		extraManifestsRefs  = flag.String("m", "", "Comma-separated list of ConfigMap names for extra manifests references")
		suppressedManifests = flag.String("s", "", "Comma-separated list of manifest names to suppress at cluster level")
		writeWarnings       = flag.Bool("w", false, "Write conversion warnings as comments to the head of converted YAML files")
		copyComments        = flag.Bool("c", false, "Copy comments from SiteConfig to ClusterInstance YAML files")
	)
	flag.Parse()

	// Get positional arguments
	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Usage: siteconfig-converter [-d output_dir] [-t cluster_namespace/name,...] [-n node_namespace/name,...] [-m configmap1,configmap2,...] [-s manifest1,manifest2,...] [-w] [-c] <siteconfig.yaml>")
		fmt.Println("\nExamples:")
		fmt.Println("  siteconfig-converter -d ./output example-siteconfig.yaml")
		fmt.Println("  siteconfig-converter -d ./output -t open-cluster-management/ai-cluster-templates-v1 -n open-cluster-management/ai-node-templates-v1 example-siteconfig.yaml")
		fmt.Println("  siteconfig-converter -t template-ns1/cluster-template1,template-ns2/cluster-template2 example-siteconfig.yaml")
		fmt.Println("  siteconfig-converter -n node-ns1/node-template1,node-ns2/node-template2 example-siteconfig.yaml")
		fmt.Println("  siteconfig-converter -m extra-manifests-cm1,extra-manifests-cm2 example-siteconfig.yaml")
		fmt.Println("  siteconfig-converter -s manifest1,manifest2 -m extra-manifests-cm1 example-siteconfig.yaml")
		fmt.Println("  siteconfig-converter -w -d ./output example-siteconfig.yaml")
		fmt.Println("  siteconfig-converter -c -d ./output example-siteconfig.yaml")
		fmt.Println("  siteconfig-converter -w -c -d ./output example-siteconfig.yaml")
		os.Exit(1)
	}

	inputFile := args[0]

	// Read the SiteConfig file
	siteConfig, err := readSiteConfig(inputFile)
	if err != nil {
		fmt.Printf("Error reading SiteConfig file: %v\n", err)
		os.Exit(1)
	}

	// Validate that it's a SiteConfig
	if siteConfig.Kind != "SiteConfig" {
		fmt.Printf("Error: File does not contain a SiteConfig (found Kind: %s)\n", siteConfig.Kind)
		os.Exit(1)
	}

	fmt.Printf("Successfully read SiteConfig: %s/%s\n", siteConfig.Metadata.Namespace, siteConfig.Metadata.Name)

	// Convert to ClusterInstance
	err = convertToClusterInstance(siteConfig, *outputDir, *clusterTemplate, *nodeTemplate, *extraManifestsRefs, *suppressedManifests, *writeWarnings, *copyComments, inputFile)
	if err != nil {
		fmt.Printf("Error converting to ClusterInstance: %v\n", err)
		os.Exit(1)
	}
}

// readSiteConfig reads and parses a SiteConfig YAML file
func readSiteConfig(filename string) (*SiteConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var siteConfig SiteConfig
	if err := yaml.Unmarshal(data, &siteConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return &siteConfig, nil
}
