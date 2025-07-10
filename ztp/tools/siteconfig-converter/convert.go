package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ClusterInstance represents the structure of a ClusterInstance CRD
type ClusterInstance struct {
	ApiVersion string                  `yaml:"apiVersion"`
	Kind       string                  `yaml:"kind"`
	Metadata   ClusterInstanceMetadata `yaml:"metadata"`
	Spec       ClusterInstanceSpec     `yaml:"spec"`
}

// ClusterInstanceMetadata represents the metadata section of a ClusterInstance
type ClusterInstanceMetadata struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

// ClusterInstanceSpec represents the spec section of a ClusterInstance
type ClusterInstanceSpec struct {
	BaseDomain             string                         `yaml:"baseDomain"`
	PullSecretRef          LocalObjectReference           `yaml:"pullSecretRef"`
	ClusterImageSetNameRef string                         `yaml:"clusterImageSetNameRef"`
	SshPublicKey           string                         `yaml:"sshPublicKey"`
	ClusterName            string                         `yaml:"clusterName"`
	ClusterType            string                         `yaml:"clusterType,omitempty"`
	NetworkType            string                         `yaml:"networkType,omitempty"`
	ApiVIPs                []string                       `yaml:"apiVIPs,omitempty"`
	IngressVIPs            []string                       `yaml:"ingressVIPs,omitempty"`
	HoldInstallation       bool                           `yaml:"holdInstallation,omitempty"`
	ClusterNetwork         []ClusterNetworkEntry          `yaml:"clusterNetwork,omitempty"`
	MachineNetwork         []MachineNetworkEntry          `yaml:"machineNetwork,omitempty"`
	ServiceNetwork         []ServiceNetworkEntry          `yaml:"serviceNetwork,omitempty"`
	AdditionalNTPSources   []string                       `yaml:"additionalNTPSources,omitempty"`
	InstallConfigOverrides string                         `yaml:"installConfigOverrides,omitempty"`
	IgnitionConfigOverride string                         `yaml:"ignitionConfigOverride,omitempty"`
	DiskEncryption         *ClusterInstanceDiskEncryption `yaml:"diskEncryption,omitempty"`
	Proxy                  *ClusterInstanceProxy          `yaml:"proxy,omitempty"`
	PlatformType           string                         `yaml:"platformType,omitempty"`
	CPUArchitecture        string                         `yaml:"cpuArchitecture,omitempty"`
	CPUPartitioningMode    string                         `yaml:"cpuPartitioningMode,omitempty"`
	ExtraAnnotations       map[string]map[string]string   `yaml:"extraAnnotations,omitempty"`
	ExtraLabels            map[string]map[string]string   `yaml:"extraLabels,omitempty"`
	ExtraManifestsRefs     []LocalObjectReference         `yaml:"extraManifestsRefs"`
	SuppressedManifests    []string                       `yaml:"suppressedManifests,omitempty"`
	PruneManifests         []ResourceRef                  `yaml:"pruneManifests,omitempty"`
	CaBundleRef            *LocalObjectReference          `yaml:"caBundleRef,omitempty"`
	Nodes                  []ClusterInstanceNode          `yaml:"nodes"`
	TemplateRefs           []TemplateRef                  `yaml:"templateRefs"`
	Reinstall              *ReinstallSpec                 `yaml:"reinstall,omitempty"`
}

// ResourceRef represents the API version and kind of a Kubernetes resource
type ResourceRef struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
}

// HostRef represents a reference to a BareMetalHost resource
type HostRef struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

// LocalObjectReference represents a reference to an object in the same namespace
type LocalObjectReference struct {
	Name string `yaml:"name"`
}

// ReinstallSpec defines the configuration for reinstallation of a ClusterInstance
type ReinstallSpec struct {
	Generation       string `yaml:"generation"`
	PreservationMode string `yaml:"preservationMode"`
}

// ClusterNetworkEntry represents a cluster network entry for ClusterInstance
type ClusterNetworkEntry struct {
	CIDR       string `yaml:"cidr"`
	HostPrefix int    `yaml:"hostPrefix,omitempty"`
}

// ServiceNetworkEntry represents a service network entry for ClusterInstance
type ServiceNetworkEntry struct {
	CIDR string `yaml:"cidr"`
}

// ClusterInstanceDiskEncryption represents disk encryption for ClusterInstance
type ClusterInstanceDiskEncryption struct {
	Type string                      `yaml:"type,omitempty"`
	Tang []ClusterInstanceTangServer `yaml:"tang,omitempty"`
}

// ClusterInstanceTangServer represents a Tang server for ClusterInstance
type ClusterInstanceTangServer struct {
	URL        string `yaml:"url"`
	Thumbprint string `yaml:"thumbprint"`
}

// ClusterInstanceProxy represents proxy settings for ClusterInstance
type ClusterInstanceProxy struct {
	HTTPProxy  string `yaml:"httpProxy,omitempty"`
	HTTPSProxy string `yaml:"httpsProxy,omitempty"`
	NoProxy    string `yaml:"noProxy,omitempty"`
}

// ClusterInstanceExtraLabels represents extra labels for ClusterInstance
type ClusterInstanceExtraLabels struct {
	ManagedCluster map[string]string `yaml:"ManagedCluster,omitempty"`
}

// ClusterInstanceNode represents a node in ClusterInstance
type ClusterInstanceNode struct {
	HostName               string                            `yaml:"hostName"`
	BmcAddress             string                            `yaml:"bmcAddress"`
	BmcCredentialsName     ClusterInstanceBmcCredentialsName `yaml:"bmcCredentialsName"`
	BootMACAddress         string                            `yaml:"bootMACAddress"`
	BootMode               string                            `yaml:"bootMode,omitempty"`
	Role                   string                            `yaml:"role,omitempty"`
	CPUArchitecture        string                            `yaml:"cpuArchitecture,omitempty"`
	RootDeviceHints        map[string]interface{}            `yaml:"rootDeviceHints,omitempty"`
	NodeNetwork            *ClusterInstanceNodeNetwork       `yaml:"nodeNetwork,omitempty"`
	IgnitionConfigOverride string                            `yaml:"ignitionConfigOverride,omitempty"`
	InstallerArgs          string                            `yaml:"installerArgs,omitempty"`
	IronicInspect          string                            `yaml:"ironicInspect,omitempty"`
	AutomatedCleaningMode  string                            `yaml:"automatedCleaningMode,omitempty"`
	NodeLabels             map[string]string                 `yaml:"nodeLabels,omitempty"`
	ExtraAnnotations       map[string]map[string]string      `yaml:"extraAnnotations,omitempty"`
	ExtraLabels            map[string]map[string]string      `yaml:"extraLabels,omitempty"`
	SuppressedManifests    []string                          `yaml:"suppressedManifests,omitempty"`
	PruneManifests         []ResourceRef                     `yaml:"pruneManifests,omitempty"`
	HostRef                *HostRef                          `yaml:"hostRef,omitempty"`
	TemplateRefs           []TemplateRef                     `yaml:"templateRefs"`
}

// ClusterInstanceBmcCredentialsName represents BMC credentials name for ClusterInstance
type ClusterInstanceBmcCredentialsName struct {
	Name string `yaml:"name"`
}

// ClusterInstanceNodeNetwork represents node network for ClusterInstance
type ClusterInstanceNodeNetwork struct {
	Config     map[string]interface{}            `yaml:"config,omitempty"`
	Interfaces []ClusterInstanceNetworkInterface `yaml:"interfaces,omitempty"`
}

// ClusterInstanceNetworkInterface represents a network interface for ClusterInstance
type ClusterInstanceNetworkInterface struct {
	Name       string `yaml:"name"`
	MacAddress string `yaml:"macAddress"`
}

// TemplateRef represents a template reference
type TemplateRef struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

// convertToClusterInstance converts a SiteConfig to ClusterInstance files
func convertToClusterInstance(siteConfig *SiteConfig, outputDir string, clusterTemplateRef string, nodeTemplateRef string, extraManifestsRefs string, suppressedManifests string) error {
	// Parse cluster template reference
	clusterParts := strings.Split(clusterTemplateRef, "/")
	if len(clusterParts) != 2 {
		return fmt.Errorf("invalid cluster template reference format, expected 'namespace/name'")
	}
	clusterTemplateNamespace, clusterTemplateName := clusterParts[0], clusterParts[1]

	// Parse node template reference
	nodeParts := strings.Split(nodeTemplateRef, "/")
	if len(nodeParts) != 2 {
		return fmt.Errorf("invalid node template reference format, expected 'namespace/name'")
	}
	nodeTemplateNamespace, nodeTemplateName := nodeParts[0], nodeParts[1]

	// Parse extra manifests refs
	var manifestsRefs []LocalObjectReference
	if extraManifestsRefs != "" {
		manifestNames := strings.Split(extraManifestsRefs, ",")
		for _, name := range manifestNames {
			name = strings.TrimSpace(name)
			if name != "" {
				manifestsRefs = append(manifestsRefs, LocalObjectReference{Name: name})
			}
		}
	}

	// Check for non-convertible fields and print warnings
	if siteConfig.Spec.SshPrivateKeySecretRef.Name != "" {
		fmt.Printf("WARNING: sshPrivateKeySecretRef field '%s' is not supported in ClusterInstance and will be ignored\n",
			siteConfig.Spec.SshPrivateKeySecretRef.Name)
	}

	// Check for global biosConfigRef
	if siteConfig.Spec.BiosConfigRef.FilePath != "" {
		fmt.Printf("WARNING: biosConfigRef field '%s' at SiteConfig spec level is not supported in ClusterInstance and will be ignored\n",
			siteConfig.Spec.BiosConfigRef.FilePath)
	}

	// Check for SiteConfig spec level crTemplates
	if len(siteConfig.Spec.CrTemplates) > 0 {
		fmt.Printf("WARNING: crTemplates field at SiteConfig spec level is not supported in ClusterInstance and will be ignored. " +
			"File-based templates are not supported. Please use ConfigMaps and reference them through templateRefs instead.\n")
	}

	// Check for cluster and node level fields
	for _, cluster := range siteConfig.Spec.Clusters {
		// Check for cluster-level biosConfigRef
		if cluster.BiosConfigRef.FilePath != "" {
			fmt.Printf("WARNING: biosConfigRef field '%s' at cluster level is not supported in ClusterInstance and will be ignored\n",
				cluster.BiosConfigRef.FilePath)
		}

		// Check for cluster-level crTemplates
		if len(cluster.CrTemplates) > 0 {
			fmt.Printf("WARNING: crTemplates field at cluster level is not supported in ClusterInstance and will be ignored. " +
				"File-based templates are not supported. Please use ConfigMaps and reference them through templateRefs instead.\n")
		}

		// Check for mergeDefaultMachineConfigs
		if cluster.MergeDefaultMachineConfigs {
			fmt.Printf("WARNING: mergeDefaultMachineConfigs field is not supported in ClusterInstance and will be ignored. " +
				"Machine config merging is not supported. Please manage machine configurations through other means.\n")
		}

		// Check for extraManifestOnly
		if cluster.ExtraManifestOnly {
			fmt.Printf("WARNING: extraManifestOnly field is not supported in ClusterInstance and will be ignored. " +
				"ClusterInstance does not support manifest-only mode. Please use standard cluster deployment.\n")
		}

		// Check for extraManifests
		if (cluster.ExtraManifests.SearchPaths != nil && len(*cluster.ExtraManifests.SearchPaths) > 0) ||
			len(cluster.ExtraManifests.Filter.Exclude) > 0 {
			fmt.Printf("WARNING: extraManifests field is not supported in ClusterInstance and will be ignored. " +
				"Directory-based manifests are not supported. Please use ConfigMaps and reference them through extraManifestsRefs instead.\n")
		}

		// Check for extraManifestPath
		if cluster.ExtraManifestPath != "" {
			fmt.Printf("WARNING: extraManifestPath field '%s' is not supported in ClusterInstance and will be ignored. "+
				"File path-based manifests are not supported. Please use ConfigMaps and reference them through extraManifestsRefs instead.\n",
				cluster.ExtraManifestPath)
		}

		// Check for siteConfigMap
		if cluster.SiteConfigMap.Name != "" {
			fmt.Printf("WARNING: siteConfigMap field '%s' is not supported in ClusterInstance and will be ignored. "+
				"Site-specific configuration maps are not supported in ClusterInstance.\n", cluster.SiteConfigMap.Name)
		}

		// Check for tpm2 in disk encryption
		if cluster.DiskEncryption.Tpm2.PCRList != "" {
			fmt.Printf("WARNING: tpm2 disk encryption configuration is not supported in ClusterInstance and will be ignored. Only Tang encryption is supported.\n")
		}

		for _, node := range cluster.Nodes {
			if len(node.DiskPartition) > 0 {
				fmt.Printf("WARNING: diskPartition field on node '%s' is not supported in ClusterInstance and will be ignored. "+
					"Consider using IgnitionConfigOverride at the node level to configure disk partitions instead.\n",
					node.HostName)
			}
			if len(node.UserData) > 0 {
				fmt.Printf("WARNING: userData field on node '%s' is not supported in ClusterInstance and will be ignored.\n",
					node.HostName)
			}
			// Check for node-level biosConfigRef
			if node.BiosConfigRef.FilePath != "" {
				fmt.Printf("WARNING: biosConfigRef field '%s' on node '%s' is not supported in ClusterInstance and will be ignored\n",
					node.BiosConfigRef.FilePath, node.HostName)
			}
			// Check for node-level crTemplates
			if len(node.CrTemplates) > 0 {
				fmt.Printf("WARNING: crTemplates field on node '%s' is not supported in ClusterInstance and will be ignored. "+
					"File-based templates are not supported. Please use ConfigMaps and reference them through templateRefs instead.\n",
					node.HostName)
			}
			// Check for cpuset
			if node.Cpuset != "" {
				fmt.Printf("WARNING: cpuset field '%s' on node '%s' is not supported in ClusterInstance and will be ignored. "+
					"Please see Workload Partitioning Feature for setting specific reserved/isolated CPUSets.\n",
					node.Cpuset, node.HostName)
			}
		}
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Convert each cluster to a ClusterInstance
	for i, cluster := range siteConfig.Spec.Clusters {
		clusterInstance := convertClusterToClusterInstance(siteConfig, cluster, clusterTemplateNamespace, clusterTemplateName, nodeTemplateNamespace, nodeTemplateName, manifestsRefs, suppressedManifests)

		// Write to file
		filename := fmt.Sprintf("%s.yaml", cluster.ClusterName)
		outputPath := filepath.Join(outputDir, filename)

		if err := writeClusterInstanceToFile(clusterInstance, outputPath); err != nil {
			return fmt.Errorf("failed to write ClusterInstance for cluster %s: %w", cluster.ClusterName, err)
		}

		fmt.Printf("Converted cluster %d (%s) to ClusterInstance: %s\n", i+1, cluster.ClusterName, outputPath)
	}

	fmt.Printf("Successfully converted %d cluster(s) to ClusterInstance files in %s\n", len(siteConfig.Spec.Clusters), outputDir)
	return nil
}

// convertClusterToClusterInstance converts a single cluster from SiteConfig to ClusterInstance
func convertClusterToClusterInstance(siteConfig *SiteConfig, cluster Cluster, clusterTemplateNamespace, clusterTemplateName, nodeTemplateNamespace, nodeTemplateName string, extraManifestsRefs []LocalObjectReference, suppressedManifests string) *ClusterInstance {
	// Determine cluster type based on number of nodes
	clusterType := "HighlyAvailable"
	if len(cluster.Nodes) == 1 {
		clusterType = "SNO"
	}

	// Convert API and Ingress VIPs to arrays
	var apiVIPs, ingressVIPs []string
	if cluster.ApiVIP != "" {
		apiVIPs = []string{cluster.ApiVIP}
	}
	if len(cluster.ApiVIPs) > 0 {
		apiVIPs = cluster.ApiVIPs
	}
	if cluster.IngressVIP != "" {
		ingressVIPs = []string{cluster.IngressVIP}
	}
	if len(cluster.IngressVIPs) > 0 {
		ingressVIPs = cluster.IngressVIPs
	}

	// Convert cluster networks
	var clusterNetworks []ClusterNetworkEntry
	for _, cn := range cluster.ClusterNetwork {
		clusterNetworks = append(clusterNetworks, ClusterNetworkEntry{
			CIDR:       cn.CIDR,
			HostPrefix: cn.HostPrefix,
		})
	}

	// Convert service networks
	var serviceNetworks []ServiceNetworkEntry
	for _, sn := range cluster.ServiceNetwork {
		serviceNetworks = append(serviceNetworks, ServiceNetworkEntry{
			CIDR: sn,
		})
	}

	// Convert disk encryption
	var diskEncryption *ClusterInstanceDiskEncryption
	if cluster.DiskEncryption.Type != "" {
		diskEncryption = &ClusterInstanceDiskEncryption{
			Type: cluster.DiskEncryption.Type,
		}
		for _, tang := range cluster.DiskEncryption.Tang {
			diskEncryption.Tang = append(diskEncryption.Tang, ClusterInstanceTangServer{
				URL:        tang.URL,
				Thumbprint: tang.Thumbprint,
			})
		}
	}

	// Convert proxy
	var proxy *ClusterInstanceProxy
	if cluster.Proxy.HTTPProxy != "" || cluster.Proxy.HTTPSProxy != "" || cluster.Proxy.NoProxy != "" {
		proxy = &ClusterInstanceProxy{
			HTTPProxy:  cluster.Proxy.HTTPProxy,
			HTTPSProxy: cluster.Proxy.HTTPSProxy,
			NoProxy:    cluster.Proxy.NoProxy,
		}
	}

	// Convert nodes
	var nodes []ClusterInstanceNode
	for _, node := range cluster.Nodes {
		// Convert node-level CrAnnotations to extraAnnotations
		var nodeExtraAnnotations map[string]map[string]string
		if len(node.CrAnnotations.Add) > 0 {
			nodeExtraAnnotations = node.CrAnnotations.Add
		}

		// Directly convert node-level CrSuppression to SuppressedManifests
		nodeSuppressedManifests := node.CrSuppression

		ciNode := ClusterInstanceNode{
			HostName:               node.HostName,
			BmcAddress:             node.BmcAddress,
			BmcCredentialsName:     ClusterInstanceBmcCredentialsName{Name: node.BmcCredentialsName.Name},
			BootMACAddress:         node.BootMACAddress,
			BootMode:               node.BootMode,
			Role:                   node.Role,
			RootDeviceHints:        node.RootDeviceHints,
			IgnitionConfigOverride: node.IgnitionConfigOverride,
			InstallerArgs:          node.InstallerArgs,
			IronicInspect:          string(node.IronicInspect),
			AutomatedCleaningMode:  node.AutomatedCleaningMode,
			NodeLabels:             node.NodeLabels,
			ExtraAnnotations:       nodeExtraAnnotations,
			SuppressedManifests:    nodeSuppressedManifests,
			TemplateRefs: []TemplateRef{
				{
					Name:      nodeTemplateName,
					Namespace: nodeTemplateNamespace,
				},
			},
		}

		// Convert node network
		if len(node.NodeNetwork.Interfaces) > 0 || len(node.NodeNetwork.Config) > 0 {
			ciNode.NodeNetwork = &ClusterInstanceNodeNetwork{
				Config: node.NodeNetwork.Config,
			}
			for _, iface := range node.NodeNetwork.Interfaces {
				ciNode.NodeNetwork.Interfaces = append(ciNode.NodeNetwork.Interfaces, ClusterInstanceNetworkInterface{
					Name:       iface.Name,
					MacAddress: iface.MacAddress,
				})
			}
		}

		nodes = append(nodes, ciNode)
	}

	// Create extra labels for ManagedCluster
	var extraLabels map[string]map[string]string
	if len(cluster.ClusterLabels) > 0 {
		extraLabels = map[string]map[string]string{
			"ManagedCluster": cluster.ClusterLabels,
		}
	}

	// Convert cluster-level CrAnnotations to extraAnnotations
	var extraAnnotations map[string]map[string]string
	if len(cluster.CrAnnotations.Add) > 0 {
		extraAnnotations = cluster.CrAnnotations.Add
	}

	// Merge extraManifestsRefs from SiteConfig with command-line provided ones
	var mergedExtraManifestsRefs []LocalObjectReference

	// Add manifestsConfigMapRefs from SiteConfig cluster
	for _, ref := range cluster.ManifestsConfigMapRefs {
		mergedExtraManifestsRefs = append(mergedExtraManifestsRefs, LocalObjectReference{Name: ref.Name})
	}

	// Add extraManifestsRefs from command line flag
	mergedExtraManifestsRefs = append(mergedExtraManifestsRefs, extraManifestsRefs...)

	// Merge cluster-level CrSuppression with suppressedManifests from command line
	var clusterSuppressedManifests []string
	clusterSuppressedManifests = append(clusterSuppressedManifests, cluster.CrSuppression...)

	// Add suppressedManifests from command line flag
	if suppressedManifests != "" {
		suppressedManifestNames := strings.Split(suppressedManifests, ",")
		for _, name := range suppressedManifestNames {
			name = strings.TrimSpace(name)
			if name != "" {
				clusterSuppressedManifests = append(clusterSuppressedManifests, name)
			}
		}
	}

	// Create ClusterInstance
	clusterInstance := &ClusterInstance{
		ApiVersion: "siteconfig.open-cluster-management.io/v1alpha1",
		Kind:       "ClusterInstance",
		Metadata: ClusterInstanceMetadata{
			Name:      cluster.ClusterName,
			Namespace: cluster.ClusterName,
		},
		Spec: ClusterInstanceSpec{
			BaseDomain:             siteConfig.Spec.BaseDomain,
			PullSecretRef:          LocalObjectReference{Name: siteConfig.Spec.PullSecretRef.Name},
			ClusterImageSetNameRef: getClusterImageSetRef(siteConfig, cluster),
			SshPublicKey:           siteConfig.Spec.SshPublicKey,
			ClusterName:            cluster.ClusterName,
			ClusterType:            clusterType,
			NetworkType:            cluster.NetworkType,
			ApiVIPs:                apiVIPs,
			IngressVIPs:            ingressVIPs,
			HoldInstallation:       cluster.HoldInstallation,
			ClusterNetwork:         clusterNetworks,
			MachineNetwork:         cluster.MachineNetwork,
			ServiceNetwork:         serviceNetworks,
			AdditionalNTPSources:   cluster.AdditionalNTPSources,
			InstallConfigOverrides: cluster.InstallConfigOverrides,
			IgnitionConfigOverride: cluster.IgnitionConfigOverride,
			DiskEncryption:         diskEncryption,
			Proxy:                  proxy,
			CPUPartitioningMode:    string(cluster.CPUPartitioningMode),
			ExtraAnnotations:       extraAnnotations,
			ExtraLabels:            extraLabels,
			ExtraManifestsRefs:     mergedExtraManifestsRefs,
			SuppressedManifests:    clusterSuppressedManifests,
			Nodes:                  nodes,
			TemplateRefs: []TemplateRef{
				{
					Name:      clusterTemplateName,
					Namespace: clusterTemplateNamespace,
				},
			},
		},
	}

	// Set optional fields only if they exist in the SiteConfig
	if cluster.PlatformType != "" {
		clusterInstance.Spec.PlatformType = cluster.PlatformType
	}
	if cluster.CPUArchitecture != "" {
		clusterInstance.Spec.CPUArchitecture = cluster.CPUArchitecture
	}

	return clusterInstance
}

// getClusterImageSetRef returns the appropriate cluster image set reference
func getClusterImageSetRef(siteConfig *SiteConfig, cluster Cluster) string {
	if cluster.ClusterImageSetNameRef != "" {
		return cluster.ClusterImageSetNameRef
	}
	return siteConfig.Spec.ClusterImageSetNameRef
}

// writeClusterInstanceToFile writes a ClusterInstance to a YAML file
func writeClusterInstanceToFile(clusterInstance *ClusterInstance, filename string) error {
	data, err := yaml.Marshal(clusterInstance)
	if err != nil {
		return fmt.Errorf("failed to marshal ClusterInstance to YAML: %w", err)
	}

	// Add YAML document separator
	content := "---\n" + string(data)

	// Add comment for extraManifestsRefs field only when it's empty
	if len(clusterInstance.Spec.ExtraManifestsRefs) == 0 {
		content = strings.Replace(content, "    extraManifestsRefs:",
			"    # extraManifestsRefs: ConfigMap references for extraManifests\n"+
				"    # Convert extraManifests to ConfigMaps and reference them here\n"+
				"    extraManifestsRefs:", 1)
	}

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
