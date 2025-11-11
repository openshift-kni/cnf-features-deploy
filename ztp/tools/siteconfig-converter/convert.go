package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// KustomizationConfigMapGeneratorSnippetFile is the filename for the generated kustomization configMapGenerator snippet
	KustomizationConfigMapGeneratorSnippetFile = "kustomization-configMapGenerator-snippet.yaml"
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
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// ClusterInstanceSpec represents the spec section of a ClusterInstance
type ClusterInstanceSpec struct {
	ClusterName            string                         `yaml:"clusterName"`
	PullSecretRef          LocalObjectReference           `yaml:"pullSecretRef"`
	ClusterImageSetNameRef string                         `yaml:"clusterImageSetNameRef"`
	SshPublicKey           string                         `yaml:"sshPublicKey"`
	BaseDomain             string                         `yaml:"baseDomain"`
	ApiVIPs                []string                       `yaml:"apiVIPs,omitempty"`
	IngressVIPs            []string                       `yaml:"ingressVIPs,omitempty"`
	HoldInstallation       bool                           `yaml:"holdInstallation,omitempty"`
	AdditionalNTPSources   []string                       `yaml:"additionalNTPSources,omitempty"`
	MachineNetwork         []MachineNetworkEntry          `yaml:"machineNetwork,omitempty"`
	ClusterNetwork         []ClusterNetworkEntry          `yaml:"clusterNetwork,omitempty"`
	ServiceNetwork         []ServiceNetworkEntry          `yaml:"serviceNetwork,omitempty"`
	NetworkType            string                         `yaml:"networkType,omitempty"`
	PlatformType           string                         `yaml:"platformType,omitempty"`
	ExtraAnnotations       map[string]map[string]string   `yaml:"extraAnnotations,omitempty"`
	ExtraLabels            map[string]map[string]string   `yaml:"extraLabels,omitempty"`
	InstallConfigOverrides string                         `yaml:"installConfigOverrides,omitempty"`
	IgnitionConfigOverride string                         `yaml:"ignitionConfigOverride,omitempty"`
	DiskEncryption         *ClusterInstanceDiskEncryption `yaml:"diskEncryption,omitempty"`
	Proxy                  *ClusterInstanceProxy          `yaml:"proxy,omitempty"`
	ExtraManifestsRefs     []LocalObjectReference         `yaml:"extraManifestsRefs"`
	SuppressedManifests    []string                       `yaml:"suppressedManifests,omitempty"`
	PruneManifests         []ResourceRef                  `yaml:"pruneManifests,omitempty"`
	CPUPartitioningMode    string                         `yaml:"cpuPartitioningMode,omitempty"`
	CPUArchitecture        string                         `yaml:"cpuArchitecture,omitempty"`
	ClusterType            string                         `yaml:"clusterType,omitempty"`
	TemplateRefs           []TemplateRef                  `yaml:"templateRefs"`
	CaBundleRef            *LocalObjectReference          `yaml:"caBundleRef,omitempty"`
	Nodes                  []ClusterInstanceNode          `yaml:"nodes"`
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
	BmcAddress             string                            `yaml:"bmcAddress"`
	BmcCredentialsName     ClusterInstanceBmcCredentialsName `yaml:"bmcCredentialsName"`
	BootMACAddress         string                            `yaml:"bootMACAddress"`
	AutomatedCleaningMode  string                            `yaml:"automatedCleaningMode,omitempty"`
	RootDeviceHints        map[string]interface{}            `yaml:"rootDeviceHints,omitempty"`
	NodeNetwork            *ClusterInstanceNodeNetwork       `yaml:"nodeNetwork,omitempty"`
	NodeLabels             map[string]string                 `yaml:"nodeLabels,omitempty"`
	HostName               string                            `yaml:"hostName"`
	HostRef                *HostRef                          `yaml:"hostRef,omitempty"`
	CPUArchitecture        string                            `yaml:"cpuArchitecture,omitempty"`
	BootMode               string                            `yaml:"bootMode,omitempty"`
	InstallerArgs          string                            `yaml:"installerArgs,omitempty"`
	IgnitionConfigOverride string                            `yaml:"ignitionConfigOverride,omitempty"`
	Role                   string                            `yaml:"role,omitempty"`
	ExtraAnnotations       map[string]map[string]string      `yaml:"extraAnnotations,omitempty"`
	ExtraLabels            map[string]map[string]string      `yaml:"extraLabels,omitempty"`
	SuppressedManifests    []string                          `yaml:"suppressedManifests,omitempty"`
	PruneManifests         []ResourceRef                     `yaml:"pruneManifests,omitempty"`
	IronicInspect          string                            `yaml:"ironicInspect,omitempty"`
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

// Kustomization represents the structure of a kustomization.yaml file
type Kustomization struct {
	ApiVersion         string               `yaml:"apiVersion"`
	Kind               string               `yaml:"kind"`
	ConfigMapGenerator []ConfigMapGenerator `yaml:"configMapGenerator"`
	GeneratorOptions   GeneratorOptions     `yaml:"generatorOptions"`
}

// ConfigMapGenerator represents a configMapGenerator entry in kustomization.yaml
type ConfigMapGenerator struct {
	Files     []string `yaml:"files"`
	Name      string   `yaml:"name"`
	Namespace string   `yaml:"namespace"`
}

// GeneratorOptions represents generatorOptions in kustomization.yaml
type GeneratorOptions struct {
	DisableNameSuffixHash bool `yaml:"disableNameSuffixHash"`
}

// WarningsCollector collects warnings during conversion
type WarningsCollector struct {
	Warnings []string
}

// AddWarning adds a warning to the collector
func (w *WarningsCollector) AddWarning(warning string) {
	w.Warnings = append(w.Warnings, warning)
}

// PrintWarnings prints all collected warnings
func (w *WarningsCollector) PrintWarnings() {
	for _, warning := range w.Warnings {
		fmt.Print(warning)
	}
}

// GenerateYAMLComments generates YAML comments from warnings
func (w *WarningsCollector) GenerateYAMLComments() string {
	if len(w.Warnings) == 0 {
		return ""
	}

	var comments strings.Builder
	comments.WriteString("# Conversion Warnings:\n")
	for _, warning := range w.Warnings {
		// Remove color codes and WARNING prefix for comments
		cleanWarning := strings.TrimPrefix(warning, "WARNING: ")
		cleanWarning = strings.TrimSpace(cleanWarning)
		if cleanWarning != "" {
			comments.WriteString("# - ")
			comments.WriteString(cleanWarning)
			comments.WriteString("\n")
		}
	}
	comments.WriteString("#\n")
	return comments.String()
}

// CommentCollector collects comments from the original SiteConfig
type CommentCollector struct {
	Comments map[string]string // Field path -> comment
}

// AddComment adds a comment for a specific field path
func (c *CommentCollector) AddComment(fieldPath, comment string) {
	if c.Comments == nil {
		c.Comments = make(map[string]string)
	}
	c.Comments[fieldPath] = comment
}

// GetComment retrieves a comment for a specific field path
func (c *CommentCollector) GetComment(fieldPath string) string {
	if c.Comments == nil {
		return ""
	}
	return c.Comments[fieldPath]
}

// parseSiteConfigWithComments parses SiteConfig YAML file preserving comments
func parseSiteConfigWithComments(filename string) (*CommentCollector, error) {
	collector := &CommentCollector{}

	// Read the file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return collector, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse with yaml.Node to preserve comments
	var node yaml.Node
	if err := yaml.Unmarshal(content, &node); err != nil {
		return collector, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Extract comments from the document
	if len(node.Content) > 0 {
		extractComments(node.Content[0], "", collector)
	}

	return collector, nil
}

// extractComments recursively extracts comments from yaml.Node
func extractComments(node *yaml.Node, path string, collector *CommentCollector) {
	if node == nil {
		return
	}

	// Add comment if it exists
	if node.HeadComment != "" {
		collector.AddComment(path, node.HeadComment)
	}
	if node.LineComment != "" {
		collector.AddComment(path+"_line", node.LineComment)
	}
	if node.FootComment != "" {
		collector.AddComment(path+"_foot", node.FootComment)
	}

	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			extractComments(child, path, collector)
		}
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]

			keyPath := path
			if keyPath != "" {
				keyPath += "."
			}
			keyPath += key.Value

			// Extract comments from key
			extractComments(key, keyPath, collector)
			// Extract comments from value
			extractComments(value, keyPath, collector)
		}
	case yaml.SequenceNode:
		for i, child := range node.Content {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			extractComments(child, childPath, collector)
		}
	}
}

// parseTemplateReferences parses a comma-separated list of template references
// Each reference should be in the format "namespace/name"
func parseTemplateReferences(templateRefs string) ([]TemplateRef, error) {
	var refs []TemplateRef

	if templateRefs == "" {
		return refs, nil
	}

	templateList := strings.Split(templateRefs, ",")
	for _, templateRef := range templateList {
		templateRef = strings.TrimSpace(templateRef)
		if templateRef == "" {
			continue
		}

		parts := strings.Split(templateRef, "/")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid template reference format '%s', expected 'namespace/name'", templateRef)
		}

		namespace := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])

		if namespace == "" || name == "" {
			return nil, fmt.Errorf("invalid template reference format '%s', namespace and name cannot be empty", templateRef)
		}

		refs = append(refs, TemplateRef{
			Name:      name,
			Namespace: namespace,
		})
	}

	return refs, nil
}

// convertToClusterInstance converts a SiteConfig to ClusterInstance files
func convertToClusterInstance(siteConfig *SiteConfig, outputDir string, clusterTemplateRef string, nodeTemplateRef string, extraManifestsRefs string, suppressedManifests string, writeWarnings bool, copyComments bool, inputFile string, extraManifestConfigMapName string) error {
	// Create warnings collector
	warningsCollector := &WarningsCollector{}

	// Create comment collector if copyComments is enabled
	var commentCollector *CommentCollector
	if copyComments {
		var err error
		commentCollector, err = parseSiteConfigWithComments(inputFile)
		if err != nil {
			fmt.Printf("Warning: Failed to parse comments from SiteConfig: %v\n", err)
			commentCollector = &CommentCollector{}
		}
	} else {
		commentCollector = &CommentCollector{}
	}

	// Parse cluster template references (comma-separated list)
	clusterTemplateRefs, err := parseTemplateReferences(clusterTemplateRef)
	if err != nil {
		return fmt.Errorf("invalid cluster template reference format: %w", err)
	}

	// Parse node template references (comma-separated list)
	nodeTemplateRefs, err := parseTemplateReferences(nodeTemplateRef)
	if err != nil {
		return fmt.Errorf("invalid node template reference format: %w", err)
	}

	// Parse extra manifests refs
	var manifestsRefs []LocalObjectReference
	if extraManifestsRefs != "" {
		manifestNames := strings.Split(extraManifestsRefs, ",")
		for _, name := range manifestNames {
			name = strings.TrimSpace(name)
			if name != "" {
				manifestsRefs = append(manifestsRefs, LocalObjectReference{Name: name})
				warningsCollector.AddWarning(fmt.Sprintf("WARNING: The specified extra manifests ConfigMap '%s' is expected to contain the correct set of manifests for the cluster which must match the content generated by the extraManifests content from the original SiteConfig. This tool can't validate that expectation\n", name))
			}
		}
	}

	// Check for non-convertible fields and print warnings
	if siteConfig.Spec.SshPrivateKeySecretRef.Name != "" {
		warningsCollector.AddWarning(fmt.Sprintf("WARNING: sshPrivateKeySecretRef field '%s' is not supported in ClusterInstance and will be ignored\n",
			siteConfig.Spec.SshPrivateKeySecretRef.Name))
	}

	// Check for global biosConfigRef
	if siteConfig.Spec.BiosConfigRef.FilePath != "" {
		warningsCollector.AddWarning(fmt.Sprintf("WARNING: biosConfigRef field '%s' at SiteConfig spec level is not supported in ClusterInstance and will be ignored. "+
			"Please create a custom node template for HostFirmwareSettings and reference it through templateRefs instead."+
			"Any nodes which use that custom template will then get the bios settings indicated in that CR\n",
			siteConfig.Spec.BiosConfigRef.FilePath))
	}

	// Check for SiteConfig spec level crTemplates
	if len(siteConfig.Spec.CrTemplates) > 0 {
		warningsCollector.AddWarning("WARNING: crTemplates field at SiteConfig spec level is not supported in ClusterInstance and will be ignored. " +
			"To provide custom CR templates please use ConfigMaps and reference them through templateRefs instead.\n")
	}

	// Check for cluster and node level fields
	for _, cluster := range siteConfig.Spec.Clusters {
		// Check for live cluster migration warnings
		if cluster.ApiVIP != "" {
			warningsCollector.AddWarning("WARNING: apiVIP is removed in ClusterInstance. " +
				"Using apiVIPs instead. If you are doing a live cluster migration, you need to create a custom template for AgentClusterInstall or suppress it.\n")
		}

		if cluster.IngressVIP != "" {
			warningsCollector.AddWarning("WARNING: ingressVIP is removed in ClusterInstance. " +
				"Using ingressVIPs instead. If you are doing a live cluster migration, you need to create a custom template for AgentClusterInstall or suppress it.\n")
		}

		// Check for cluster-level biosConfigRef
		if cluster.BiosConfigRef.FilePath != "" {
			warningsCollector.AddWarning(fmt.Sprintf("WARNING: biosConfigRef field '%s' at cluster level is not supported in ClusterInstance and will be ignored. "+
				"Please create a custom node template for HostFirmwareSettings and reference it through templateRefs instead."+
				"Any nodes which use that custom template will then get the bios settings indicated in that CR\n",
				cluster.BiosConfigRef.FilePath))
		}

		// Check for cluster-level crTemplates
		if len(cluster.CrTemplates) > 0 {
			warningsCollector.AddWarning("WARNING: crTemplates field at cluster level is not supported in ClusterInstance and will be ignored. " +
				"To provide custom CR templates please use ConfigMaps and reference them through templateRefs instead.\n")
		}

		// Check for mergeDefaultMachineConfigs
		if cluster.MergeDefaultMachineConfigs {
			warningsCollector.AddWarning("WARNING: mergeDefaultMachineConfigs field is not supported in ClusterInstance and will be ignored. " +
				"Use a ConfigMap which contains the already merged MachineConfigs and reference it through extraManifestsRefs instead.\n")
		}

		if cluster.ExtraManifestOnly {
			warningsCollector.AddWarning("WARNING: extraManifestOnly field is not part of ClusterInstance spec. " +
				"Extra manifests will be generated from this SiteConfig and included in the extraManifestsRefs ConfigMap, but the full ClusterInstance CR set will also be generated.\n")
		}

		if cluster.ExtraManifestPath != "" {
			warningsCollector.AddWarning(fmt.Sprintf("WARNING: extraManifestPath field '%s' is not supported in ClusterInstance and will be ignored. "+
				"Use extraManifests field instead\n",
				cluster.ExtraManifestPath))
		}

		// Check for siteConfigMap
		if cluster.SiteConfigMap.Name != "" {
			warningsCollector.AddWarning(fmt.Sprintf("WARNING: siteConfigMap field '%s' is not supported in ClusterInstance and will be ignored. "+
				"Create the site specific ConfigMap and place in git as a separate resource.\n", cluster.SiteConfigMap.Name))
		}

		// Check for tpm2 in disk encryption
		if cluster.DiskEncryption.Tpm2.PCRList != "" {
			warningsCollector.AddWarning("WARNING: tpm2 disk encryption configuration is not supported in ClusterInstance and will be ignored. Conversion will be done only for the Tang server field." +
				"disk encryption MachineConfig with correct parameters must be added directly to the extramanifests configmap\n")
		}

		for _, node := range cluster.Nodes {
			if len(node.DiskPartition) > 0 {
				warningsCollector.AddWarning(fmt.Sprintf("WARNING: diskPartition field on node '%s' is not supported in ClusterInstance and will be ignored. "+
					"Consider using IgnitionConfigOverride at the node level to configure disk partitions instead.\n",
					node.HostName))
			}
			if len(node.UserData) > 0 {
				warningsCollector.AddWarning(fmt.Sprintf("WARNING: userData field on node '%s' is not supported in ClusterInstance and will be ignored."+
					"Add userData through custom templates which add the necessary field to BareMetalHost\n",
					node.HostName))
			}
			// Check for node-level biosConfigRef
			if node.BiosConfigRef.FilePath != "" {
				warningsCollector.AddWarning(fmt.Sprintf("WARNING: biosConfigRef field '%s' on node '%s' is not supported in ClusterInstance and will be ignored. "+
					"Please create a custom node template that includes HostFirmwareSettings and reference it through templateRefs instead."+
					"Any nodes which use that custom template will then get the bios settings indicated in that CR\n",
					node.BiosConfigRef.FilePath, node.HostName))
			}
			// Check for node-level crTemplates
			if len(node.CrTemplates) > 0 {
				warningsCollector.AddWarning(fmt.Sprintf("WARNING: crTemplates field on node '%s' is not supported in ClusterInstance and will be ignored. "+
					"To provide custom CR templates please use ConfigMaps and reference them through templateRefs instead.\n",
					node.HostName))
			}
			// Check for cpuset
			if node.Cpuset != "" {
				warningsCollector.AddWarning(fmt.Sprintf("WARNING: cpuset field '%s' on node '%s' is not supported in ClusterInstance and will be ignored. "+
					"Please see Workload Partitioning Feature for setting specific reserved/isolated CPUSets.\n",
					node.Cpuset, node.HostName))
			}
		}
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Convert each cluster to a ClusterInstance
	var generatedFiles []string
	for i, cluster := range siteConfig.Spec.Clusters {
		clusterInstance := convertClusterToClusterInstance(siteConfig, cluster, clusterTemplateRefs, nodeTemplateRefs, manifestsRefs, suppressedManifests, warningsCollector, i, filepath.Base(inputFile), extraManifestConfigMapName)

		// Write to file
		filename := fmt.Sprintf("%s.yaml", cluster.ClusterName)
		outputPath := filepath.Join(outputDir, filename)
		generatedFiles = append(generatedFiles, filename)

		if err := writeClusterInstanceToFile(clusterInstance, outputPath, warningsCollector, writeWarnings, commentCollector, copyComments, i, cluster); err != nil {
			return fmt.Errorf("failed to write ClusterInstance for cluster %s: %w", cluster.ClusterName, err)
		}

		fmt.Printf("Converted cluster %d (%s) to ClusterInstance: %s\n", i+1, cluster.ClusterName, outputPath)
	}

	// Print warnings to console if not writing to file
	if !writeWarnings {
		warningsCollector.PrintWarnings()
	}

	// Create the success message with file names
	var successMessage string
	if len(generatedFiles) == 1 {
		successMessage = fmt.Sprintf("Successfully converted %d cluster(s) to ClusterInstance files in %s: %s", len(siteConfig.Spec.Clusters), outputDir, generatedFiles[0])
	} else {
		successMessage = fmt.Sprintf("Successfully converted %d cluster(s) to ClusterInstance files in %s: %s", len(siteConfig.Spec.Clusters), outputDir, strings.Join(generatedFiles, ", "))
	}
	fmt.Println(successMessage)
	return nil
}

// insertSiteConfigComments inserts comments from SiteConfig into ClusterInstance YAML
func insertSiteConfigComments(content string, commentCollector *CommentCollector, clusterIndex int, cluster Cluster) string {
	if commentCollector == nil || commentCollector.Comments == nil {
		return content
	}

	// Map SiteConfig field paths to ClusterInstance field paths (dynamic based on cluster index)
	fieldMapping := map[string]string{
		"spec.baseDomain":             "spec.baseDomain",
		"spec.pullSecretRef":          "spec.pullSecretRef",
		"spec.clusterImageSetNameRef": "spec.clusterImageSetNameRef",
		"spec.sshPublicKey":           "spec.sshPublicKey",
		fmt.Sprintf("spec.clusters[%d].clusterName", clusterIndex):            "spec.clusterName",
		fmt.Sprintf("spec.clusters[%d].networkType", clusterIndex):            "spec.networkType",
		fmt.Sprintf("spec.clusters[%d].clusterNetwork", clusterIndex):         "spec.clusterNetwork",
		fmt.Sprintf("spec.clusters[%d].machineNetwork", clusterIndex):         "spec.machineNetwork",
		fmt.Sprintf("spec.clusters[%d].serviceNetwork", clusterIndex):         "spec.serviceNetwork",
		fmt.Sprintf("spec.clusters[%d].additionalNTPSources", clusterIndex):   "spec.additionalNTPSources",
		fmt.Sprintf("spec.clusters[%d].apiVIP", clusterIndex):                 "spec.apiVIP",
		fmt.Sprintf("spec.clusters[%d].ingressVIP", clusterIndex):             "spec.ingressVIP",
		fmt.Sprintf("spec.clusters[%d].apiVIPs", clusterIndex):                "spec.apiVIPs",
		fmt.Sprintf("spec.clusters[%d].ingressVIPs", clusterIndex):            "spec.ingressVIPs",
		fmt.Sprintf("spec.clusters[%d].holdInstallation", clusterIndex):       "spec.holdInstallation",
		fmt.Sprintf("spec.clusters[%d].installConfigOverrides", clusterIndex): "spec.installConfigOverrides",
		fmt.Sprintf("spec.clusters[%d].ignitionConfigOverride", clusterIndex): "spec.ignitionConfigOverride",
		fmt.Sprintf("spec.clusters[%d].diskEncryption", clusterIndex):         "spec.diskEncryption",
		fmt.Sprintf("spec.clusters[%d].proxy", clusterIndex):                  "spec.proxy",
		fmt.Sprintf("spec.clusters[%d].cpuPartitioningMode", clusterIndex):    "spec.cpuPartitioningMode",
		fmt.Sprintf("spec.clusters[%d].nodes", clusterIndex):                  "spec.nodes",
	}

	// Add node-level field mappings dynamically
	nodeFieldMappings := map[string]string{
		"hostName":               "hostName",
		"bmcAddress":             "bmcAddress",
		"bmcCredentialsName":     "bmcCredentialsName",
		"bootMACAddress":         "bootMACAddress",
		"bootMode":               "bootMode",
		"role":                   "role",
		"nodeLabels":             "nodeLabels",
		"nodeNetwork":            "nodeNetwork",
		"ignitionConfigOverride": "ignitionConfigOverride",
		"installerArgs":          "installerArgs",
		"ironicInspect":          "ironicInspect",
		"templateRefs":           "templateRefs",
	}

	for i := 0; i < len(cluster.Nodes); i++ {
		for siteConfigNodeField, clusterInstanceNodeField := range nodeFieldMappings {
			siteConfigPath := fmt.Sprintf("spec.clusters[%d].nodes[%d].%s", clusterIndex, i, siteConfigNodeField)
			clusterInstancePath := fmt.Sprintf("spec.nodes[%d].%s", i, clusterInstanceNodeField)
			fieldMapping[siteConfigPath] = clusterInstancePath
		}
	}

	// Collect unmapped comments for header section
	usedComments := make(map[string]bool)
	headerComments := []string{}

	// First pass: collect all comments that are not node-specific and not already mapped
	for path, comment := range commentCollector.Comments {
		// Skip node-specific comments (will be handled separately if needed)
		if strings.Contains(path, "nodes[") {
			continue
		}

		// Check if this comment is mapped to a field
		isMapped := false
		for siteConfigPath := range fieldMapping {
			if path == siteConfigPath || path == siteConfigPath+"_line" || path == siteConfigPath+"_foot" {
				isMapped = true
				break
			}
		}

		// If not mapped, add to header comments
		if !isMapped && comment != "" {
			cleanComment := strings.TrimSpace(comment)
			if strings.HasPrefix(cleanComment, "#") {
				cleanComment = strings.TrimSpace(cleanComment[1:])
			}
			if cleanComment != "" {
				headerComments = append(headerComments, cleanComment)
			}
		}
	}

	// Add header comments if any exist
	if len(headerComments) > 0 {
		lines := strings.Split(content, "\n")
		var result []string

		// Add document separator
		result = append(result, "---")

		// Add header comments section
		result = append(result, "# Comments from original SiteConfig:")
		for _, comment := range headerComments {
			result = append(result, "# "+comment)
		}
		result = append(result, "#")

		// Add the rest of the content (skip the original "---" line)
		for i, line := range lines {
			if i == 0 && line == "---" {
				continue // Skip the original document separator
			}
			result = append(result, line)
		}

		content = strings.Join(result, "\n")
	}

	// Second pass: add field-specific comments
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		// Add the original line
		result = append(result, line)

		// Check if this line matches any of our field mappings
		for siteConfigPath, clusterInstancePath := range fieldMapping {
			// Check if line contains the field
			fieldName := getFieldName(clusterInstancePath)
			if strings.Contains(line, fieldName+":") {
				// Look for comments associated with this field
				comment := commentCollector.GetComment(siteConfigPath)
				if comment == "" {
					comment = commentCollector.GetComment(siteConfigPath + "_line")
				}
				if comment != "" && !usedComments[siteConfigPath] {
					// Mark as used to avoid duplicates
					usedComments[siteConfigPath] = true

					// Insert comment before the field
					indent := getIndentation(line)
					commentLines := strings.Split(strings.TrimSpace(comment), "\n")

					// Find where to insert the comment (before the current line)
					insertIndex := len(result) - 1
					originalLine := result[insertIndex]

					// Remove the original line temporarily
					result = result[:insertIndex]

					// Add all comment lines
					for _, commentLine := range commentLines {
						if commentLine != "" {
							// Clean up the comment line (remove leading # if present)
							cleanComment := strings.TrimSpace(commentLine)
							if strings.HasPrefix(cleanComment, "#") {
								cleanComment = strings.TrimSpace(cleanComment[1:])
							}
							result = append(result, indent+"# "+cleanComment)
						}
					}

					// Add the original line back
					result = append(result, originalLine)
				}
			}
		}
	}

	return strings.Join(result, "\n")
}

// getFieldName extracts the field name from a field path
func getFieldName(fieldPath string) string {
	parts := strings.Split(fieldPath, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fieldPath
}

// getIndentation returns the indentation of a line
func getIndentation(line string) string {
	for i, char := range line {
		if char != ' ' && char != '\t' {
			return line[:i]
		}
	}
	return line
}

// convertClusterToClusterInstance converts a single cluster from SiteConfig to ClusterInstance
func convertClusterToClusterInstance(siteConfig *SiteConfig, cluster Cluster, clusterTemplateRefs, nodeTemplateRefs []TemplateRef, extraManifestsRefs []LocalObjectReference, suppressedManifests string, warningsCollector *WarningsCollector, clusterIndex int, sourceFilename string, extraManifestConfigMapName string) *ClusterInstance {
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

	if len(cluster.ApiVIPs) > 0 && cluster.ApiVIP != "" && cluster.ApiVIP != cluster.ApiVIPs[0] {
		warningsCollector.AddWarning("WARNING: apiVIP must be the same as the first element of apiVIPs")
	}

	if cluster.IngressVIP != "" {
		ingressVIPs = []string{cluster.IngressVIP}
	}
	if len(cluster.IngressVIPs) > 0 {
		ingressVIPs = cluster.IngressVIPs
	}

	if len(cluster.IngressVIPs) > 0 && cluster.IngressVIP != "" &&
		cluster.IngressVIP != cluster.IngressVIPs[0] {
		warningsCollector.AddWarning("WARNING: IngressVIP must be the same as the first element of IngressVIPs")
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
			TemplateRefs:           nodeTemplateRefs,
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

		if ciNode.IronicInspect == string(InspectEnabled) {
			ciNode.IronicInspect = ""
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

	warningsCollector.AddWarning(fmt.Sprintf("WARNING: Added default extraManifest ConfigMap '%s' to extraManifestsRefs. This configmap is created automatically.\n", extraManifestConfigMapName))
	mergedExtraManifestsRefs = append(mergedExtraManifestsRefs, LocalObjectReference{Name: extraManifestConfigMapName})

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
			Annotations: map[string]string{
				"siteconfig-converter": fmt.Sprintf("from %s at %s", sourceFilename, time.Now().Format(time.RFC3339)),
			},
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
			TemplateRefs:           clusterTemplateRefs,
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
func writeClusterInstanceToFile(clusterInstance *ClusterInstance, filename string, warningsCollector *WarningsCollector, writeWarnings bool, commentCollector *CommentCollector, copyComments bool, clusterIndex int, cluster Cluster) error {
	// Use yaml.Encoder with 2-space indentation instead of yaml.Marshal
	var buf strings.Builder
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2) // Set indentation to 2 spaces

	err := encoder.Encode(clusterInstance)
	if err != nil {
		return fmt.Errorf("failed to marshal ClusterInstance to YAML: %w", err)
	}
	encoder.Close()

	// Add YAML document separator
	content := "---\n" + buf.String()

	// Add warnings as comments at the head of the file if writeWarnings is true
	if writeWarnings && len(warningsCollector.Warnings) > 0 {
		warningsComments := warningsCollector.GenerateYAMLComments()
		content = "---\n" + warningsComments + buf.String()
	}

	// Add comments from original SiteConfig if copyComments is enabled
	if copyComments {
		content = insertSiteConfigComments(content, commentCollector, clusterIndex, cluster)
	}

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func generateKustomizationYAML(configMapName, configMapNamespace, manifestsDir, outputDir string) error {
	fmt.Println("--- Kustomization.yaml Generator ---")

	// Validate if the directory exists
	if _, err := os.Stat(manifestsDir); os.IsNotExist(err) {
		return fmt.Errorf("directory '%s' not found", manifestsDir)
	}

	// Get absolute path for debugging
	absPath, err := filepath.Abs(manifestsDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	fmt.Printf("Scanning directory: %s\n", absPath)

	// Find all .yaml and .yml files in the specified directory (top-level only, no subdirectories)
	var yamlFiles []string
	entries, err := os.ReadDir(manifestsDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext == ".yaml" || ext == ".yml" {
				// Get relative path from outputDir to the manifest file
				relPath, err := filepath.Rel(outputDir, filepath.Join(manifestsDir, entry.Name()))
				if err != nil {
					return fmt.Errorf("failed to get relative path for %s: %w", entry.Name(), err)
				}
				yamlFiles = append(yamlFiles, relPath)
				fmt.Printf("Found and adding: %s\n", relPath)
			}
		}
	}

	if len(yamlFiles) == 0 {
		fmt.Printf("No .yaml or .yml files found in '%s'. Generating kustomization.yaml with empty files list.\n", manifestsDir)
	}

	// Sort files alphanumerically
	sort.Strings(yamlFiles)

	// Create the kustomization struct
	kustomization := Kustomization{
		ApiVersion: "kustomize.config.k8s.io/v1beta1",
		Kind:       "Kustomization",
		ConfigMapGenerator: []ConfigMapGenerator{
			{
				Files:     yamlFiles,
				Name:      configMapName,
				Namespace: configMapNamespace,
			},
		},
		GeneratorOptions: GeneratorOptions{
			DisableNameSuffixHash: true,
		},
	}

	// Marshal to YAML
	kustomizationData, err := yaml.Marshal(kustomization)
	if err != nil {
		return fmt.Errorf("failed to marshal kustomization to YAML: %w", err)
	}

	kustomizationContent := string(kustomizationData)

	outputFile := filepath.Join(outputDir, KustomizationConfigMapGeneratorSnippetFile)
	if err := os.WriteFile(outputFile, []byte(kustomizationContent), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", KustomizationConfigMapGeneratorSnippetFile, err)
	}

	fmt.Println("------------------------------------")
	fmt.Printf("%s generated successfully at: %s\n", KustomizationConfigMapGeneratorSnippetFile, outputFile)
	fmt.Println("Content:")
	fmt.Println(kustomizationContent)
	fmt.Println("------------------------------------")

	return nil
}
