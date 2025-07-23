package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// Helper function to create template references for tests
func createTemplateRefs(namespace, name string) []TemplateRef {
	return []TemplateRef{
		{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func TestConvertSNOSiteConfig(t *testing.T) {
	// Read the test SNO SiteConfig file
	siteConfig, err := readSiteConfig("samples/test-sno-siteconfig.yaml")
	if err != nil {
		t.Fatalf("Failed to read test SNO SiteConfig: %v", err)
	}

	// Validate basic SiteConfig properties
	if siteConfig.Kind != "SiteConfig" {
		t.Errorf("Expected Kind to be 'SiteConfig', got '%s'", siteConfig.Kind)
	}

	if siteConfig.Metadata.Name != "example-sno" {
		t.Errorf("Expected metadata name to be 'example-sno', got '%s'", siteConfig.Metadata.Name)
	}

	if len(siteConfig.Spec.Clusters) != 1 {
		t.Fatalf("Expected 1 cluster, got %d", len(siteConfig.Spec.Clusters))
	}

	cluster := siteConfig.Spec.Clusters[0]

	// Test conversion
	clusterTemplateNamespace := "test-namespace"
	clusterTemplateName := "test-template"
	nodeTemplateNamespace := "open-cluster-management"
	nodeTemplateName := "ai-node-templates-v1"
	clusterInstance := convertClusterToClusterInstance(siteConfig, cluster, createTemplateRefs(clusterTemplateNamespace, clusterTemplateName), createTemplateRefs(nodeTemplateNamespace, nodeTemplateName), []LocalObjectReference{}, "", &WarningsCollector{}, 0, "test-siteconfig.yaml")

	// Test basic ClusterInstance properties
	if clusterInstance.ApiVersion != "siteconfig.open-cluster-management.io/v1alpha1" {
		t.Errorf("Expected apiVersion to be 'siteconfig.open-cluster-management.io/v1alpha1', got '%s'", clusterInstance.ApiVersion)
	}

	if clusterInstance.Kind != "ClusterInstance" {
		t.Errorf("Expected kind to be 'ClusterInstance', got '%s'", clusterInstance.Kind)
	}

	// Test metadata
	if clusterInstance.Metadata.Name != "example-sno" {
		t.Errorf("Expected metadata name to be 'example-sno', got '%s'", clusterInstance.Metadata.Name)
	}

	if clusterInstance.Metadata.Namespace != "example-sno" {
		t.Errorf("Expected metadata namespace to be 'example-sno', got '%s'", clusterInstance.Metadata.Namespace)
	}

	// Test cluster type detection for SNO
	if clusterInstance.Spec.ClusterType != "SNO" {
		t.Errorf("Expected clusterType to be 'SNO' for single node, got '%s'", clusterInstance.Spec.ClusterType)
	}

	// Test cpuPartitioningMode preservation
	if clusterInstance.Spec.CPUPartitioningMode != "AllNodes" {
		t.Errorf("Expected cpuPartitioningMode to be 'AllNodes', got '%s'", clusterInstance.Spec.CPUPartitioningMode)
	}

	// Test basic spec fields
	if clusterInstance.Spec.BaseDomain != "example.com" {
		t.Errorf("Expected baseDomain to be 'example.com', got '%s'", clusterInstance.Spec.BaseDomain)
	}

	if clusterInstance.Spec.ClusterName != "example-sno" {
		t.Errorf("Expected clusterName to be 'example-sno', got '%s'", clusterInstance.Spec.ClusterName)
	}

	if clusterInstance.Spec.NetworkType != "OVNKubernetes" {
		t.Errorf("Expected networkType to be 'OVNKubernetes', got '%s'", clusterInstance.Spec.NetworkType)
	}

	// Test cluster network preservation
	if len(clusterInstance.Spec.ClusterNetwork) != 1 {
		t.Errorf("Expected 1 cluster network, got %d", len(clusterInstance.Spec.ClusterNetwork))
	} else {
		cn := clusterInstance.Spec.ClusterNetwork[0]
		if cn.CIDR != "1001:1::/48" {
			t.Errorf("Expected cluster network CIDR to be '1001:1::/48', got '%s'", cn.CIDR)
		}
		if cn.HostPrefix != 64 {
			t.Errorf("Expected cluster network host prefix to be 64, got %d", cn.HostPrefix)
		}
	}

	// Test service network preservation
	if len(clusterInstance.Spec.ServiceNetwork) != 1 {
		t.Errorf("Expected 1 service network, got %d", len(clusterInstance.Spec.ServiceNetwork))
	} else {
		sn := clusterInstance.Spec.ServiceNetwork[0]
		if sn.CIDR != "1001:2::/112" {
			t.Errorf("Expected service network CIDR to be '1001:2::/112', got '%s'", sn.CIDR)
		}
	}

	// Test machine network preservation
	if len(clusterInstance.Spec.MachineNetwork) != 1 {
		t.Errorf("Expected 1 machine network, got %d", len(clusterInstance.Spec.MachineNetwork))
	} else {
		mn := clusterInstance.Spec.MachineNetwork[0]
		if mn.CIDR != "1111:2222:3333:4444::/64" {
			t.Errorf("Expected machine network CIDR to be '1111:2222:3333:4444::/64', got '%s'", mn.CIDR)
		}
	}

	// Test NTP sources preservation
	if len(clusterInstance.Spec.AdditionalNTPSources) != 1 {
		t.Errorf("Expected 1 NTP source, got %d", len(clusterInstance.Spec.AdditionalNTPSources))
	} else {
		ntp := clusterInstance.Spec.AdditionalNTPSources[0]
		if ntp != "1111:2222:3333:4444::2" {
			t.Errorf("Expected NTP source to be '1111:2222:3333:4444::2', got '%s'", ntp)
		}
	}

	// Test cluster labels preservation in extraLabels.ManagedCluster
	if clusterInstance.Spec.ExtraLabels == nil {
		t.Error("Expected extraLabels to be set, but it was nil")
	} else {
		managedClusterLabels, exists := clusterInstance.Spec.ExtraLabels["ManagedCluster"]
		if !exists {
			t.Error("Expected extraLabels.ManagedCluster to be set, but it was not found")
		} else {
			expectedLabels := map[string]string{
				"du-profile":   "latest",
				"common":       "true",
				"group-du-sno": "",
				"sites":        "example-sno",
			}

			for key, expectedValue := range expectedLabels {
				if actualValue, exists := managedClusterLabels[key]; !exists {
					t.Errorf("Expected cluster label '%s' to exist in extraLabels.ManagedCluster", key)
				} else if actualValue != expectedValue {
					t.Errorf("Expected cluster label '%s' to be '%s', got '%s'", key, expectedValue, actualValue)
				}
			}
		}
	}

	// Test node configuration
	if len(clusterInstance.Spec.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(clusterInstance.Spec.Nodes))
	}

	node := clusterInstance.Spec.Nodes[0]

	// Test node basic properties
	if node.HostName != "example-node1.example.com" {
		t.Errorf("Expected node hostname to be 'example-node1.example.com', got '%s'", node.HostName)
	}

	if node.Role != "master" {
		t.Errorf("Expected node role to be 'master', got '%s'", node.Role)
	}

	if node.BmcAddress != "idrac-virtualmedia+https://[1111:2222:3333:4444::bbbb:1]/redfish/v1/Systems/System.Embedded.1" {
		t.Errorf("Expected correct BMC address, got '%s'", node.BmcAddress)
	}

	if node.BootMACAddress != "AA:BB:CC:DD:EE:11" {
		t.Errorf("Expected boot MAC address to be 'AA:BB:CC:DD:EE:11', got '%s'", node.BootMACAddress)
	}

	if node.BootMode != "UEFISecureBoot" {
		t.Errorf("Expected boot mode to be 'UEFISecureBoot', got '%s'", node.BootMode)
	}

	// Test node network configuration
	if node.NodeNetwork == nil {
		t.Fatal("Expected node network configuration to exist")
	}

	if len(node.NodeNetwork.Interfaces) != 1 {
		t.Errorf("Expected 1 network interface, got %d", len(node.NodeNetwork.Interfaces))
	} else {
		iface := node.NodeNetwork.Interfaces[0]
		if iface.Name != "eno1" {
			t.Errorf("Expected interface name to be 'eno1', got '%s'", iface.Name)
		}
		if iface.MacAddress != "AA:BB:CC:DD:EE:11" {
			t.Errorf("Expected interface MAC address to be 'AA:BB:CC:DD:EE:11', got '%s'", iface.MacAddress)
		}
	}

	// Test ignition config override preservation
	if node.IgnitionConfigOverride == "" {
		t.Error("Expected ignitionConfigOverride to be preserved and not empty")
	} else {
		// Check that it contains expected ignition version
		if !strings.Contains(node.IgnitionConfigOverride, `"version": "3.2.0"`) {
			t.Error("Expected ignitionConfigOverride to contain ignition version 3.2.0")
		}
	}

	// Test template references
	if len(node.TemplateRefs) != 1 {
		t.Errorf("Expected 1 template reference, got %d", len(node.TemplateRefs))
	} else {
		template := node.TemplateRefs[0]
		if template.Name != nodeTemplateName {
			t.Errorf("Expected template name to be '%s', got '%s'", nodeTemplateName, template.Name)
		}
		if template.Namespace != nodeTemplateNamespace {
			t.Errorf("Expected template namespace to be '%s', got '%s'", nodeTemplateNamespace, template.Namespace)
		}
	}

	// Test platform type and CPU architecture are optional (should be empty if not specified in SiteConfig)
	if clusterInstance.Spec.PlatformType != "" {
		t.Errorf("Expected platformType to be empty since not specified in SiteConfig, got '%s'", clusterInstance.Spec.PlatformType)
	}

	if clusterInstance.Spec.CPUArchitecture != "" {
		t.Errorf("Expected cpuArchitecture to be empty since not specified in SiteConfig, got '%s'", clusterInstance.Spec.CPUArchitecture)
	}
}

func TestConvert3NodeSiteConfig(t *testing.T) {
	// Load the 3-node SiteConfig
	siteConfig, err := readSiteConfig("samples/test-3node-siteconfig.yaml")
	if err != nil {
		t.Fatalf("Failed to read test 3-node SiteConfig: %v", err)
	}

	// Validate basic SiteConfig properties
	if siteConfig.Kind != "SiteConfig" {
		t.Errorf("Expected Kind to be 'SiteConfig', got '%s'", siteConfig.Kind)
	}

	if siteConfig.Metadata.Name != "example-3node" {
		t.Errorf("Expected metadata name to be 'example-3node', got '%s'", siteConfig.Metadata.Name)
	}

	if len(siteConfig.Spec.Clusters) != 1 {
		t.Fatalf("Expected 1 cluster, got %d", len(siteConfig.Spec.Clusters))
	}

	cluster := siteConfig.Spec.Clusters[0]

	// Test conversion
	clusterTemplateNamespace := "test-namespace"
	clusterTemplateName := "test-template"
	nodeTemplateNamespace := "open-cluster-management"
	nodeTemplateName := "ai-node-templates-v1"
	clusterInstance := convertClusterToClusterInstance(siteConfig, cluster, createTemplateRefs(clusterTemplateNamespace, clusterTemplateName), createTemplateRefs(nodeTemplateNamespace, nodeTemplateName), []LocalObjectReference{}, "", &WarningsCollector{}, 0, "test-siteconfig.yaml")

	// Test basic ClusterInstance properties
	if clusterInstance.Kind != "ClusterInstance" {
		t.Errorf("Expected kind to be 'ClusterInstance', got '%s'", clusterInstance.Kind)
	}

	if clusterInstance.Metadata.Name != "example-3node" {
		t.Errorf("Expected metadata name to be 'example-3node', got '%s'", clusterInstance.Metadata.Name)
	}

	// Test cluster type is HighlyAvailable for 3-node
	if clusterInstance.Spec.ClusterType != "HighlyAvailable" {
		t.Errorf("Expected cluster type 'HighlyAvailable' for 3-node cluster, got '%s'", clusterInstance.Spec.ClusterType)
	}

	// Test IPv6 configuration
	if len(clusterInstance.Spec.ClusterNetwork) != 1 {
		t.Errorf("Expected 1 cluster network, got %d", len(clusterInstance.Spec.ClusterNetwork))
	} else {
		cn := clusterInstance.Spec.ClusterNetwork[0]
		if cn.CIDR != "1001:1::/48" {
			t.Errorf("Expected cluster network CIDR '1001:1::/48', got '%s'", cn.CIDR)
		}
		if cn.HostPrefix != 64 {
			t.Errorf("Expected cluster network host prefix 64, got %d", cn.HostPrefix)
		}
	}

	// Test VIPs
	if len(clusterInstance.Spec.ApiVIPs) != 1 {
		t.Errorf("Expected 1 API VIP, got %d", len(clusterInstance.Spec.ApiVIPs))
	} else {
		if clusterInstance.Spec.ApiVIPs[0] != "1111:2222:3333:4444::1:1" {
			t.Errorf("Expected API VIP '1111:2222:3333:4444::1:1', got '%s'", clusterInstance.Spec.ApiVIPs[0])
		}
	}

	if len(clusterInstance.Spec.IngressVIPs) != 1 {
		t.Errorf("Expected 1 Ingress VIP, got %d", len(clusterInstance.Spec.IngressVIPs))
	} else {
		if clusterInstance.Spec.IngressVIPs[0] != "1111:2222:3333:4444::1:2" {
			t.Errorf("Expected Ingress VIP '1111:2222:3333:4444::1:2', got '%s'", clusterInstance.Spec.IngressVIPs[0])
		}
	}

	// Test service network
	if len(clusterInstance.Spec.ServiceNetwork) != 1 {
		t.Errorf("Expected 1 service network, got %d", len(clusterInstance.Spec.ServiceNetwork))
	} else {
		sn := clusterInstance.Spec.ServiceNetwork[0]
		if sn.CIDR != "1001:2::/112" {
			t.Errorf("Expected service network CIDR '1001:2::/112', got '%s'", sn.CIDR)
		}
	}

	// Test cluster labels in extraLabels.ManagedCluster
	expectedLabels := map[string]string{
		"du-profile":     "latest",
		"common":         "true",
		"group-du-3node": "",
		"sites":          "example-multinode",
	}

	if clusterInstance.Spec.ExtraLabels == nil {
		t.Error("Expected extraLabels to be set")
	} else {
		managedClusterLabels, exists := clusterInstance.Spec.ExtraLabels["ManagedCluster"]
		if !exists {
			t.Error("Expected extraLabels.ManagedCluster to be set, but it was not found")
		} else {
			for key, expectedValue := range expectedLabels {
				if actualValue, exists := managedClusterLabels[key]; !exists {
					t.Errorf("Expected label '%s' not found in extraLabels.ManagedCluster", key)
				} else if actualValue != expectedValue {
					t.Errorf("Expected label '%s' to be '%s', got '%s'", key, expectedValue, actualValue)
				}
			}
		}
	}

	// Test nodes
	if len(clusterInstance.Spec.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(clusterInstance.Spec.Nodes))
	}

	// Test all nodes are masters
	for i, node := range clusterInstance.Spec.Nodes {
		if node.Role != "master" {
			t.Errorf("Expected node %d to have role 'master', got '%s'", i, node.Role)
		}
		if node.BootMode != "UEFISecureBoot" {
			t.Errorf("Expected node %d to have boot mode 'UEFISecureBoot', got '%s'", i, node.BootMode)
		}
		if deviceName, ok := node.RootDeviceHints["deviceName"]; !ok || deviceName != "/dev/disk/by-path/pci-0000:01:00.0-scsi-0:2:0:0" {
			t.Errorf("Expected node %d to have specific root device hint", i)
		}
	}

	// Test first node specifically
	firstNode := clusterInstance.Spec.Nodes[0]
	if firstNode.HostName != "example-node1.example.com" {
		t.Errorf("Expected first node hostname 'example-node1.example.com', got '%s'", firstNode.HostName)
	}
	if firstNode.BmcAddress != "idrac-virtualmedia+https://[1111:2222:3333:4444::bbbb:1]/redfish/v1/Systems/System.Embedded.1" {
		t.Errorf("Expected first node BMC address to match expected format")
	}
	if firstNode.BootMACAddress != "AA:BB:CC:DD:EE:11" {
		t.Errorf("Expected first node boot MAC address 'AA:BB:CC:DD:EE:11', got '%s'", firstNode.BootMACAddress)
	}

	// Test node labels for first node
	if firstNode.NodeLabels == nil {
		t.Error("Expected first node to have node labels")
	} else {
		if _, exists := firstNode.NodeLabels["node-role.kubernetes.io/master-du"]; !exists {
			t.Error("Expected first node to have master-du label")
		}
		if firstNode.NodeLabels["custom-label/parameter1"] != "true" {
			t.Error("Expected first node to have custom-label/parameter1 set to 'true'")
		}
	}

	// Test network configuration
	if firstNode.NodeNetwork.Interfaces == nil || len(firstNode.NodeNetwork.Interfaces) != 1 {
		t.Error("Expected first node to have 1 network interface")
	} else {
		iface := firstNode.NodeNetwork.Interfaces[0]
		if iface.Name != "eno1" {
			t.Error("Expected first node interface name to be 'eno1'")
		}
		if iface.MacAddress != "AA:BB:CC:DD:EE:11" {
			t.Error("Expected first node interface MAC address to match")
		}
	}
}

func TestConvert5NodeSiteConfig(t *testing.T) {
	// Load the 5-node SiteConfig
	siteConfig, err := readSiteConfig("samples/test-5node-siteconfig.yaml")
	if err != nil {
		t.Fatalf("Failed to read test 5-node SiteConfig: %v", err)
	}

	// Validate basic SiteConfig properties
	if siteConfig.Kind != "SiteConfig" {
		t.Errorf("Expected Kind to be 'SiteConfig', got '%s'", siteConfig.Kind)
	}

	if siteConfig.Metadata.Name != "example-standard" {
		t.Errorf("Expected metadata name to be 'example-standard', got '%s'", siteConfig.Metadata.Name)
	}

	if len(siteConfig.Spec.Clusters) != 1 {
		t.Fatalf("Expected 1 cluster, got %d", len(siteConfig.Spec.Clusters))
	}

	cluster := siteConfig.Spec.Clusters[0]

	// Test conversion
	clusterTemplateNamespace := "test-namespace"
	clusterTemplateName := "test-template"
	nodeTemplateNamespace := "open-cluster-management"
	nodeTemplateName := "ai-node-templates-v1"
	clusterInstance := convertClusterToClusterInstance(siteConfig, cluster, createTemplateRefs(clusterTemplateNamespace, clusterTemplateName), createTemplateRefs(nodeTemplateNamespace, nodeTemplateName), []LocalObjectReference{}, "", &WarningsCollector{}, 0, "test-siteconfig.yaml")

	// Test basic ClusterInstance properties
	if clusterInstance.Kind != "ClusterInstance" {
		t.Errorf("Expected kind to be 'ClusterInstance', got '%s'", clusterInstance.Kind)
	}

	if clusterInstance.Metadata.Name != "example-standard" {
		t.Errorf("Expected metadata name to be 'example-standard', got '%s'", clusterInstance.Metadata.Name)
	}

	// Test cluster type is HighlyAvailable for 5-node (3 masters + 2 workers)
	if clusterInstance.Spec.ClusterType != "HighlyAvailable" {
		t.Errorf("Expected cluster type 'HighlyAvailable' for 5-node cluster, got '%s'", clusterInstance.Spec.ClusterType)
	}

	// Test IPv6 configuration
	if len(clusterInstance.Spec.ClusterNetwork) != 1 {
		t.Errorf("Expected 1 cluster network, got %d", len(clusterInstance.Spec.ClusterNetwork))
	} else {
		cn := clusterInstance.Spec.ClusterNetwork[0]
		if cn.CIDR != "1001:1::/48" {
			t.Errorf("Expected cluster network CIDR '1001:1::/48', got '%s'", cn.CIDR)
		}
		if cn.HostPrefix != 64 {
			t.Errorf("Expected cluster network host prefix 64, got %d", cn.HostPrefix)
		}
	}

	// Test VIPs
	if len(clusterInstance.Spec.ApiVIPs) != 1 {
		t.Errorf("Expected 1 API VIP, got %d", len(clusterInstance.Spec.ApiVIPs))
	} else {
		if clusterInstance.Spec.ApiVIPs[0] != "1111:2222:3333:4444::1:1" {
			t.Errorf("Expected API VIP '1111:2222:3333:4444::1:1', got '%s'", clusterInstance.Spec.ApiVIPs[0])
		}
	}

	if len(clusterInstance.Spec.IngressVIPs) != 1 {
		t.Errorf("Expected 1 Ingress VIP, got %d", len(clusterInstance.Spec.IngressVIPs))
	} else {
		if clusterInstance.Spec.IngressVIPs[0] != "1111:2222:3333:4444::1:2" {
			t.Errorf("Expected Ingress VIP '1111:2222:3333:4444::1:2', got '%s'", clusterInstance.Spec.IngressVIPs[0])
		}
	}

	// Test service network
	if len(clusterInstance.Spec.ServiceNetwork) != 1 {
		t.Errorf("Expected 1 service network, got %d", len(clusterInstance.Spec.ServiceNetwork))
	} else {
		sn := clusterInstance.Spec.ServiceNetwork[0]
		if sn.CIDR != "1001:2::/112" {
			t.Errorf("Expected service network CIDR '1001:2::/112', got '%s'", sn.CIDR)
		}
	}

	// Test cluster labels in extraLabels.ManagedCluster
	expectedLabels := map[string]string{
		"du-profile":        "latest",
		"common":            "true",
		"group-du-standard": "",
		"sites":             "example-multinode",
	}

	if clusterInstance.Spec.ExtraLabels == nil {
		t.Error("Expected extraLabels to be set")
	} else {
		managedClusterLabels, exists := clusterInstance.Spec.ExtraLabels["ManagedCluster"]
		if !exists {
			t.Error("Expected extraLabels.ManagedCluster to be set, but it was not found")
		} else {
			for key, expectedValue := range expectedLabels {
				if actualValue, exists := managedClusterLabels[key]; !exists {
					t.Errorf("Expected label '%s' not found in extraLabels.ManagedCluster", key)
				} else if actualValue != expectedValue {
					t.Errorf("Expected label '%s' to be '%s', got '%s'", key, expectedValue, actualValue)
				}
			}
		}
	}

	// Test nodes
	if len(clusterInstance.Spec.Nodes) != 5 {
		t.Errorf("Expected 5 nodes, got %d", len(clusterInstance.Spec.Nodes))
	}

	// Test master nodes (first 3)
	for i := 0; i < 3; i++ {
		node := clusterInstance.Spec.Nodes[i]
		if node.Role != "master" {
			t.Errorf("Expected node %d to have role 'master', got '%s'", i, node.Role)
		}
		if node.BootMode != "UEFISecureBoot" {
			t.Errorf("Expected node %d to have boot mode 'UEFISecureBoot', got '%s'", i, node.BootMode)
		}
		if deviceName, ok := node.RootDeviceHints["deviceName"]; !ok || deviceName != "/dev/disk/by-path/pci-0000:01:00.0-scsi-0:2:0:0" {
			t.Errorf("Expected node %d to have specific root device hint", i)
		}
	}

	// Test worker nodes (last 2)
	for i := 3; i < 5; i++ {
		node := clusterInstance.Spec.Nodes[i]
		if node.Role != "worker" {
			t.Errorf("Expected node %d to have role 'worker', got '%s'", i, node.Role)
		}
		if node.BootMode != "UEFISecureBoot" {
			t.Errorf("Expected node %d to have boot mode 'UEFISecureBoot', got '%s'", i, node.BootMode)
		}
		if deviceName, ok := node.RootDeviceHints["deviceName"]; !ok || deviceName != "/dev/disk/by-path/pci-0000:01:00.0-scsi-0:2:0:0" {
			t.Errorf("Expected node %d to have specific root device hint", i)
		}
	}

	// Test first node specifically
	firstNode := clusterInstance.Spec.Nodes[0]
	if firstNode.HostName != "example-node1.example.com" {
		t.Errorf("Expected first node hostname 'example-node1.example.com', got '%s'", firstNode.HostName)
	}
	if firstNode.BmcAddress != "idrac-virtualmedia+https://[1111:2222:3333:4444::bbbb:1]/redfish/v1/Systems/System.Embedded.1" {
		t.Errorf("Expected first node BMC address to match expected format")
	}
	if firstNode.BootMACAddress != "AA:BB:CC:DD:EE:11" {
		t.Errorf("Expected first node boot MAC address 'AA:BB:CC:DD:EE:11', got '%s'", firstNode.BootMACAddress)
	}

	// Test worker node specifically (node4)
	workerNode := clusterInstance.Spec.Nodes[3]
	if workerNode.HostName != "example-node4.example.com" {
		t.Errorf("Expected worker node hostname 'example-node4.example.com', got '%s'", workerNode.HostName)
	}
	if workerNode.BmcAddress != "idrac-virtualmedia+https://[1111:2222:3333:4444::bbbb:4]/redfish/v1/Systems/System.Embedded.1" {
		t.Errorf("Expected worker node BMC address to match expected format")
	}
	if workerNode.BootMACAddress != "AA:BB:CC:DD:EE:44" {
		t.Errorf("Expected worker node boot MAC address 'AA:BB:CC:DD:EE:44', got '%s'", workerNode.BootMACAddress)
	}
	if workerNode.Role != "worker" {
		t.Errorf("Expected worker node role 'worker', got '%s'", workerNode.Role)
	}

	// Test network configuration
	if firstNode.NodeNetwork.Interfaces == nil || len(firstNode.NodeNetwork.Interfaces) != 1 {
		t.Error("Expected first node to have 1 network interface")
	} else {
		iface := firstNode.NodeNetwork.Interfaces[0]
		if iface.Name != "eno1" {
			t.Error("Expected first node interface name to be 'eno1'")
		}
		if iface.MacAddress != "AA:BB:CC:DD:EE:11" {
			t.Error("Expected first node interface MAC address to match")
		}
	}
}

func TestComprehensiveFieldConversion(t *testing.T) {
	// Read the actual example-sno1.yaml file
	siteConfig, err := readSiteConfig("samples/example-sno1.yaml")
	if err != nil {
		t.Fatalf("Failed to read example-sno1.yaml: %v", err)
	}

	// Validate that it's a SiteConfig
	if siteConfig.Kind != "SiteConfig" {
		t.Fatalf("Expected Kind to be 'SiteConfig', got '%s'", siteConfig.Kind)
	}

	if len(siteConfig.Spec.Clusters) != 1 {
		t.Fatalf("Expected 1 cluster in example-sno1.yaml, got %d", len(siteConfig.Spec.Clusters))
	}

	// Convert to ClusterInstance
	cluster := siteConfig.Spec.Clusters[0]
	clusterTemplateNamespace := "test-cluster-namespace"
	clusterTemplateName := "test-cluster-template"
	nodeTemplateNamespace := "test-node-namespace"
	nodeTemplateName := "test-node-template"

	clusterInstance := convertClusterToClusterInstance(siteConfig, cluster, createTemplateRefs(clusterTemplateNamespace, clusterTemplateName), createTemplateRefs(nodeTemplateNamespace, nodeTemplateName), []LocalObjectReference{}, "", &WarningsCollector{}, 0, "test-siteconfig.yaml")

	// Verify API Version and Kind
	if clusterInstance.ApiVersion != "siteconfig.open-cluster-management.io/v1alpha1" {
		t.Errorf("Expected apiVersion 'siteconfig.open-cluster-management.io/v1alpha1', got '%s'", clusterInstance.ApiVersion)
	}
	if clusterInstance.Kind != "ClusterInstance" {
		t.Errorf("Expected kind 'ClusterInstance', got '%s'", clusterInstance.Kind)
	}

	// Verify Metadata
	if clusterInstance.Metadata.Name != "sno1" {
		t.Errorf("Expected metadata name 'sno1', got '%s'", clusterInstance.Metadata.Name)
	}
	if clusterInstance.Metadata.Namespace != "sno1" {
		t.Errorf("Expected metadata namespace 'sno1', got '%s'", clusterInstance.Metadata.Namespace)
	}

	// Verify Spec - Basic Fields
	if clusterInstance.Spec.BaseDomain != "5g-deployment.lab" {
		t.Errorf("Expected baseDomain '5g-deployment.lab', got '%s'", clusterInstance.Spec.BaseDomain)
	}
	if clusterInstance.Spec.PullSecretRef.Name != "disconnected-registry-pull-secret" {
		t.Errorf("Expected pullSecretRef name 'disconnected-registry-pull-secret', got '%s'", clusterInstance.Spec.PullSecretRef.Name)
	}
	if clusterInstance.Spec.ClusterImageSetNameRef != "active-ocp-version" {
		t.Errorf("Expected clusterImageSetNameRef 'active-ocp-version', got '%s'", clusterInstance.Spec.ClusterImageSetNameRef)
	}
	if clusterInstance.Spec.SshPublicKey != "ssh-rsa REDACTED" {
		t.Errorf("Expected sshPublicKey 'ssh-rsa REDACTED', got '%s'", clusterInstance.Spec.SshPublicKey)
	}
	if clusterInstance.Spec.ClusterName != "sno1" {
		t.Errorf("Expected clusterName 'sno1', got '%s'", clusterInstance.Spec.ClusterName)
	}
	if clusterInstance.Spec.ClusterType != "SNO" {
		t.Errorf("Expected clusterType 'SNO' for single node, got '%s'", clusterInstance.Spec.ClusterType)
	}
	if clusterInstance.Spec.NetworkType != "OVNKubernetes" {
		t.Errorf("Expected networkType 'OVNKubernetes', got '%s'", clusterInstance.Spec.NetworkType)
	}

	// Verify VIPs - example-sno1.yaml doesn't have VIPs set, so they should be empty
	if len(clusterInstance.Spec.ApiVIPs) != 0 {
		t.Errorf("Expected 0 API VIPs, got %d", len(clusterInstance.Spec.ApiVIPs))
	}
	if len(clusterInstance.Spec.IngressVIPs) != 0 {
		t.Errorf("Expected 0 Ingress VIPs, got %d", len(clusterInstance.Spec.IngressVIPs))
	}

	// Verify HoldInstallation
	if clusterInstance.Spec.HoldInstallation != false {
		t.Errorf("Expected holdInstallation false, got %t", clusterInstance.Spec.HoldInstallation)
	}

	// Verify Networks - example-sno1.yaml has IPv4 networks
	if len(clusterInstance.Spec.ClusterNetwork) != 1 {
		t.Errorf("Expected 1 cluster network, got %d", len(clusterInstance.Spec.ClusterNetwork))
	} else {
		if clusterInstance.Spec.ClusterNetwork[0].CIDR != "10.128.0.0/14" {
			t.Errorf("Expected cluster network[0] CIDR '10.128.0.0/14', got '%s'", clusterInstance.Spec.ClusterNetwork[0].CIDR)
		}
		if clusterInstance.Spec.ClusterNetwork[0].HostPrefix != 23 {
			t.Errorf("Expected cluster network[0] hostPrefix 23, got %d", clusterInstance.Spec.ClusterNetwork[0].HostPrefix)
		}
	}

	if len(clusterInstance.Spec.MachineNetwork) != 1 {
		t.Errorf("Expected 1 machine network, got %d", len(clusterInstance.Spec.MachineNetwork))
	} else {
		if clusterInstance.Spec.MachineNetwork[0].CIDR != "192.168.125.0/24" {
			t.Errorf("Expected machine network[0] CIDR '192.168.125.0/24', got '%s'", clusterInstance.Spec.MachineNetwork[0].CIDR)
		}
	}

	if len(clusterInstance.Spec.ServiceNetwork) != 1 {
		t.Errorf("Expected 1 service network, got %d", len(clusterInstance.Spec.ServiceNetwork))
	} else {
		if clusterInstance.Spec.ServiceNetwork[0].CIDR != "172.30.0.0/16" {
			t.Errorf("Expected service network[0] CIDR '172.30.0.0/16', got '%s'", clusterInstance.Spec.ServiceNetwork[0].CIDR)
		}
	}

	// Verify NTP Sources
	expectedNTPSources := []string{"infra.5g-deployment.lab"}
	if len(clusterInstance.Spec.AdditionalNTPSources) != len(expectedNTPSources) {
		t.Errorf("Expected %d NTP sources, got %d", len(expectedNTPSources), len(clusterInstance.Spec.AdditionalNTPSources))
	}
	for i, expected := range expectedNTPSources {
		if i < len(clusterInstance.Spec.AdditionalNTPSources) && clusterInstance.Spec.AdditionalNTPSources[i] != expected {
			t.Errorf("Expected NTP source[%d] '%s', got '%s'", i, expected, clusterInstance.Spec.AdditionalNTPSources[i])
		}
	}

	// Verify Override Configs
	if clusterInstance.Spec.InstallConfigOverrides != `{"capabilities":{"baselineCapabilitySet": "None", "additionalEnabledCapabilities": [ "OperatorLifecycleManager", "NodeTuning", "Ingress" ] }}` {
		t.Errorf("Expected installConfigOverrides preserved, got '%s'", clusterInstance.Spec.InstallConfigOverrides)
	}
	// example-sno1.yaml doesn't have ignition config override, so it should be empty
	if clusterInstance.Spec.IgnitionConfigOverride != "" {
		t.Errorf("Expected ignitionConfigOverride to be empty, got '%s'", clusterInstance.Spec.IgnitionConfigOverride)
	}

	// Verify Disk Encryption - example-sno1.yaml doesn't have disk encryption
	if clusterInstance.Spec.DiskEncryption != nil {
		t.Error("Expected diskEncryption to be nil (not set in example-sno1.yaml)")
	}

	// Verify Proxy - example-sno1.yaml doesn't have proxy settings
	if clusterInstance.Spec.Proxy != nil {
		t.Error("Expected proxy to be nil (not set in example-sno1.yaml)")
	}

	// Verify Platform and CPU - example-sno1.yaml doesn't have these set
	if clusterInstance.Spec.PlatformType != "" {
		t.Errorf("Expected platformType to be empty, got '%s'", clusterInstance.Spec.PlatformType)
	}
	if clusterInstance.Spec.CPUArchitecture != "" {
		t.Errorf("Expected cpuArchitecture to be empty, got '%s'", clusterInstance.Spec.CPUArchitecture)
	}
	if clusterInstance.Spec.CPUPartitioningMode != "AllNodes" {
		t.Errorf("Expected cpuPartitioningMode 'AllNodes', got '%s'", clusterInstance.Spec.CPUPartitioningMode)
	}

	// Verify ExtraAnnotations - example-sno1.yaml doesn't have CrAnnotations
	if clusterInstance.Spec.ExtraAnnotations != nil {
		t.Error("Expected extraAnnotations to be nil (not set in example-sno1.yaml)")
	}

	// Verify ExtraLabels - example-sno1.yaml has cluster labels
	if clusterInstance.Spec.ExtraLabels == nil {
		t.Error("Expected extraLabels to be set")
	} else {
		if managedClusterLabels, exists := clusterInstance.Spec.ExtraLabels["ManagedCluster"]; !exists {
			t.Error("Expected extraLabels.ManagedCluster to exist")
		} else {
			expectedLabels := map[string]string{
				"common":        "ocp418",
				"logicalGroup":  "active",
				"group-du-sno":  "",
				"du-site":       "sno1",
				"du-zone":       "europe",
				"hardware-type": "hw-type-platform-1",
			}
			for key, expectedValue := range expectedLabels {
				if actualValue, exists := managedClusterLabels[key]; !exists {
					t.Errorf("Expected extraLabels.ManagedCluster['%s'] to exist", key)
				} else if actualValue != expectedValue {
					t.Errorf("Expected extraLabels.ManagedCluster['%s'] = '%s', got '%s'", key, expectedValue, actualValue)
				}
			}
		}
	}

	// Verify SuppressedManifests - example-sno1.yaml doesn't have CrSuppression
	if len(clusterInstance.Spec.SuppressedManifests) != 0 {
		t.Errorf("Expected 0 suppressed manifests, got %d", len(clusterInstance.Spec.SuppressedManifests))
	}

	// Verify Template References
	if len(clusterInstance.Spec.TemplateRefs) != 1 {
		t.Errorf("Expected 1 template reference, got %d", len(clusterInstance.Spec.TemplateRefs))
	} else {
		if clusterInstance.Spec.TemplateRefs[0].Name != clusterTemplateName {
			t.Errorf("Expected template name '%s', got '%s'", clusterTemplateName, clusterInstance.Spec.TemplateRefs[0].Name)
		}
		if clusterInstance.Spec.TemplateRefs[0].Namespace != clusterTemplateNamespace {
			t.Errorf("Expected template namespace '%s', got '%s'", clusterTemplateNamespace, clusterInstance.Spec.TemplateRefs[0].Namespace)
		}
	}

	// Verify Nodes
	if len(clusterInstance.Spec.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(clusterInstance.Spec.Nodes))
	}

	node := clusterInstance.Spec.Nodes[0]

	// Verify Node Basic Fields
	if node.HostName != "sno1.5g-deployment.lab" {
		t.Errorf("Expected node hostName 'sno1.5g-deployment.lab', got '%s'", node.HostName)
	}
	if node.Role != "master" {
		t.Errorf("Expected node role 'master', got '%s'", node.Role)
	}
	if node.BmcAddress != "redfish-virtualmedia://192.168.125.1:9000/redfish/v1/Systems/local/sno1" {
		t.Errorf("Expected node bmcAddress 'redfish-virtualmedia://192.168.125.1:9000/redfish/v1/Systems/local/sno1', got '%s'", node.BmcAddress)
	}
	if node.BmcCredentialsName.Name != "sno1-bmc-credentials" {
		t.Errorf("Expected node bmcCredentialsName 'sno1-bmc-credentials', got '%s'", node.BmcCredentialsName.Name)
	}
	if node.BootMACAddress != "AA:AA:AA:AA:02:01" {
		t.Errorf("Expected node bootMACAddress 'AA:AA:AA:AA:02:01', got '%s'", node.BootMACAddress)
	}
	if node.BootMode != "UEFI" {
		t.Errorf("Expected node bootMode 'UEFI', got '%s'", node.BootMode)
	}
	if node.AutomatedCleaningMode != "" {
		t.Errorf("Expected node automatedCleaningMode to be empty, got '%s'", node.AutomatedCleaningMode)
	}
	if node.InstallerArgs != "" {
		t.Errorf("Expected node installerArgs to be empty, got '%s'", node.InstallerArgs)
	}
	if node.IronicInspect != "" {
		t.Errorf("Expected node ironicInspect to be empty, got '%s'", node.IronicInspect)
	}

	// Verify Node Root Device Hints
	if node.RootDeviceHints == nil {
		t.Error("Expected node rootDeviceHints to be set")
	} else {
		if deviceName, exists := node.RootDeviceHints["deviceName"]; !exists {
			t.Error("Expected node rootDeviceHints.deviceName to exist")
		} else if deviceName != "/dev/vda" {
			t.Errorf("Expected node rootDeviceHints.deviceName '/dev/vda', got '%v'", deviceName)
		}
		// example-sno1.yaml doesn't have minSizeGigabytes
		if _, exists := node.RootDeviceHints["minSizeGigabytes"]; exists {
			t.Error("Expected node rootDeviceHints.minSizeGigabytes to not exist in example-sno1.yaml")
		}
	}

	// Verify Node Labels (should NOT have extraLabels.node)
	if node.NodeLabels == nil {
		t.Error("Expected node nodeLabels to be set")
	} else {
		expectedNodeLabels := map[string]string{
			"5gran.lab/my-custom-label": "",
		}
		for key, expectedValue := range expectedNodeLabels {
			if actualValue, exists := node.NodeLabels[key]; !exists {
				t.Errorf("Expected node nodeLabels['%s'] to exist", key)
			} else if actualValue != expectedValue {
				t.Errorf("Expected node nodeLabels['%s'] = '%s', got '%s'", key, expectedValue, actualValue)
			}
		}
	}

	// Verify Node ExtraLabels should NOT be set (this was the fix)
	if node.ExtraLabels != nil {
		t.Error("Expected node extraLabels to be nil (should not duplicate nodeLabels)")
	}

	// Verify Node Ignition Config Override - example-sno1.yaml doesn't have node-level ignition config
	if node.IgnitionConfigOverride != "" {
		t.Errorf("Expected node ignitionConfigOverride to be empty, got '%s'", node.IgnitionConfigOverride)
	}

	// Verify Node ExtraAnnotations - example-sno1.yaml has node-level crAnnotations
	if node.ExtraAnnotations == nil {
		t.Error("Expected node extraAnnotations to be set (from crAnnotations in example-sno1.yaml)")
	} else {
		if bmhAnnotations, exists := node.ExtraAnnotations["BareMetalHost"]; !exists {
			t.Error("Expected node extraAnnotations['BareMetalHost'] to exist")
		} else {
			expectedAnnotation := "bmac.agent-install.openshift.io/remove-agent-and-node-on-delete"
			if value, exists := bmhAnnotations[expectedAnnotation]; !exists {
				t.Errorf("Expected node extraAnnotations['BareMetalHost']['%s'] to exist", expectedAnnotation)
			} else if value != "true" {
				t.Errorf("Expected node extraAnnotations['BareMetalHost']['%s'] = 'true', got '%s'", expectedAnnotation, value)
			}
		}
	}

	// Verify Node SuppressedManifests - example-sno1.yaml has node-level crSuppression
	if len(node.SuppressedManifests) != 1 {
		t.Errorf("Expected 1 node suppressed manifest, got %d", len(node.SuppressedManifests))
	} else {
		if node.SuppressedManifests[0] != "BareMetalHost" {
			t.Errorf("Expected node suppressedManifests[0] = 'BareMetalHost', got '%s'", node.SuppressedManifests[0])
		}
	}

	// Verify Node Network
	if node.NodeNetwork == nil {
		t.Error("Expected node nodeNetwork to be set")
	} else {
		if node.NodeNetwork.Config == nil {
			t.Error("Expected node nodeNetwork.config to be set")
		}
		if len(node.NodeNetwork.Interfaces) != 1 {
			t.Errorf("Expected 1 node network interface, got %d", len(node.NodeNetwork.Interfaces))
		} else {
			if node.NodeNetwork.Interfaces[0].Name != "enp3s0" {
				t.Errorf("Expected node network interface name 'enp3s0', got '%s'", node.NodeNetwork.Interfaces[0].Name)
			}
			if node.NodeNetwork.Interfaces[0].MacAddress != "AA:AA:AA:AA:02:01" {
				t.Errorf("Expected node network interface MAC 'AA:AA:AA:AA:02:01', got '%s'", node.NodeNetwork.Interfaces[0].MacAddress)
			}
		}
	}

	// Verify Node Template References
	if len(node.TemplateRefs) != 1 {
		t.Errorf("Expected 1 node template reference, got %d", len(node.TemplateRefs))
	} else {
		if node.TemplateRefs[0].Name != nodeTemplateName {
			t.Errorf("Expected node template name '%s', got '%s'", nodeTemplateName, node.TemplateRefs[0].Name)
		}
		if node.TemplateRefs[0].Namespace != nodeTemplateNamespace {
			t.Errorf("Expected node template namespace '%s', got '%s'", nodeTemplateNamespace, node.TemplateRefs[0].Namespace)
		}
	}

	// Verify default extraManifestsRefs (should contain cluster name when no manifests provided)
	if len(clusterInstance.Spec.ExtraManifestsRefs) != 1 {
		t.Errorf("Expected extraManifestsRefs to contain 1 default entry, got %d", len(clusterInstance.Spec.ExtraManifestsRefs))
	} else if clusterInstance.Spec.ExtraManifestsRefs[0].Name != "sno1" {
		t.Errorf("Expected default extraManifestsRefs[0].Name to be 'sno1', got '%s'", clusterInstance.Spec.ExtraManifestsRefs[0].Name)
	}
	if clusterInstance.Spec.CaBundleRef != nil {
		t.Error("Expected caBundleRef to be nil (not set in test SiteConfig)")
	}
	if clusterInstance.Spec.Reinstall != nil {
		t.Error("Expected reinstall to be nil (not set in test SiteConfig)")
	}
	if len(clusterInstance.Spec.PruneManifests) > 0 {
		t.Error("Expected pruneManifests to be empty (not set in test SiteConfig)")
	}
	if len(node.PruneManifests) > 0 {
		t.Error("Expected node pruneManifests to be empty (not set in test SiteConfig)")
	}
	if node.HostRef != nil {
		t.Error("Expected node hostRef to be nil (not set in test SiteConfig)")
	}
	if node.CPUArchitecture != "" {
		t.Error("Expected node cpuArchitecture to be empty (not set in test SiteConfig)")
	}
}

func TestComprehensive3NodeFieldConversion(t *testing.T) {
	// Read the actual test-3node-siteconfig.yaml file
	siteConfig, err := readSiteConfig("samples/test-3node-siteconfig.yaml")
	if err != nil {
		t.Fatalf("Failed to read test-3node-siteconfig.yaml: %v", err)
	}

	// Validate that it's a SiteConfig
	if siteConfig.Kind != "SiteConfig" {
		t.Fatalf("Expected Kind to be 'SiteConfig', got '%s'", siteConfig.Kind)
	}

	if len(siteConfig.Spec.Clusters) != 1 {
		t.Fatalf("Expected 1 cluster in test-3node-siteconfig.yaml, got %d", len(siteConfig.Spec.Clusters))
	}

	// Convert to ClusterInstance
	cluster := siteConfig.Spec.Clusters[0]
	clusterTemplateNamespace := "test-cluster-namespace"
	clusterTemplateName := "test-cluster-template"
	nodeTemplateNamespace := "test-node-namespace"
	nodeTemplateName := "test-node-template"

	clusterInstance := convertClusterToClusterInstance(siteConfig, cluster, createTemplateRefs(clusterTemplateNamespace, clusterTemplateName), createTemplateRefs(nodeTemplateNamespace, nodeTemplateName), []LocalObjectReference{}, "", &WarningsCollector{}, 0, "test-siteconfig.yaml")

	// Verify API Version and Kind
	if clusterInstance.ApiVersion != "siteconfig.open-cluster-management.io/v1alpha1" {
		t.Errorf("Expected apiVersion 'siteconfig.open-cluster-management.io/v1alpha1', got '%s'", clusterInstance.ApiVersion)
	}
	if clusterInstance.Kind != "ClusterInstance" {
		t.Errorf("Expected kind 'ClusterInstance', got '%s'", clusterInstance.Kind)
	}

	// Verify Metadata
	if clusterInstance.Metadata.Name != "example-3node" {
		t.Errorf("Expected metadata name 'example-3node', got '%s'", clusterInstance.Metadata.Name)
	}
	if clusterInstance.Metadata.Namespace != "example-3node" {
		t.Errorf("Expected metadata namespace 'example-3node', got '%s'", clusterInstance.Metadata.Namespace)
	}

	// Verify Spec - Basic Fields
	if clusterInstance.Spec.BaseDomain != "example.com" {
		t.Errorf("Expected baseDomain 'example.com', got '%s'", clusterInstance.Spec.BaseDomain)
	}
	if clusterInstance.Spec.PullSecretRef.Name != "assisted-deployment-pull-secret" {
		t.Errorf("Expected pullSecretRef name 'assisted-deployment-pull-secret', got '%s'", clusterInstance.Spec.PullSecretRef.Name)
	}
	if clusterInstance.Spec.ClusterImageSetNameRef != "openshift-4.19" {
		t.Errorf("Expected clusterImageSetNameRef 'openshift-4.19', got '%s'", clusterInstance.Spec.ClusterImageSetNameRef)
	}
	if clusterInstance.Spec.SshPublicKey != "ssh-rsa AAAA..." {
		t.Errorf("Expected sshPublicKey 'ssh-rsa AAAA...', got '%s'", clusterInstance.Spec.SshPublicKey)
	}
	if clusterInstance.Spec.ClusterName != "example-3node" {
		t.Errorf("Expected clusterName 'example-3node', got '%s'", clusterInstance.Spec.ClusterName)
	}
	if clusterInstance.Spec.ClusterType != "HighlyAvailable" {
		t.Errorf("Expected clusterType 'HighlyAvailable' for 3-node cluster, got '%s'", clusterInstance.Spec.ClusterType)
	}
	if clusterInstance.Spec.NetworkType != "OVNKubernetes" {
		t.Errorf("Expected networkType 'OVNKubernetes', got '%s'", clusterInstance.Spec.NetworkType)
	}

	// Verify VIPs - test-3node-siteconfig.yaml has IPv6 VIPs from ApiVIP/IngressVIP fields
	expectedAPIVIPs := []string{"1111:2222:3333:4444::1:1"}
	if len(clusterInstance.Spec.ApiVIPs) != len(expectedAPIVIPs) {
		t.Errorf("Expected %d API VIPs, got %d", len(expectedAPIVIPs), len(clusterInstance.Spec.ApiVIPs))
	}
	for i, expected := range expectedAPIVIPs {
		if i < len(clusterInstance.Spec.ApiVIPs) && clusterInstance.Spec.ApiVIPs[i] != expected {
			t.Errorf("Expected API VIP[%d] '%s', got '%s'", i, expected, clusterInstance.Spec.ApiVIPs[i])
		}
	}

	expectedIngressVIPs := []string{"1111:2222:3333:4444::1:2"}
	if len(clusterInstance.Spec.IngressVIPs) != len(expectedIngressVIPs) {
		t.Errorf("Expected %d Ingress VIPs, got %d", len(expectedIngressVIPs), len(clusterInstance.Spec.IngressVIPs))
	}
	for i, expected := range expectedIngressVIPs {
		if i < len(clusterInstance.Spec.IngressVIPs) && clusterInstance.Spec.IngressVIPs[i] != expected {
			t.Errorf("Expected Ingress VIP[%d] '%s', got '%s'", i, expected, clusterInstance.Spec.IngressVIPs[i])
		}
	}

	// Verify HoldInstallation - should be false by default
	if clusterInstance.Spec.HoldInstallation != false {
		t.Errorf("Expected holdInstallation false, got %t", clusterInstance.Spec.HoldInstallation)
	}

	// Verify Networks - test-3node-siteconfig.yaml has IPv6 networks
	if len(clusterInstance.Spec.ClusterNetwork) != 1 {
		t.Errorf("Expected 1 cluster network, got %d", len(clusterInstance.Spec.ClusterNetwork))
	} else {
		if clusterInstance.Spec.ClusterNetwork[0].CIDR != "1001:1::/48" {
			t.Errorf("Expected cluster network[0] CIDR '1001:1::/48', got '%s'", clusterInstance.Spec.ClusterNetwork[0].CIDR)
		}
		if clusterInstance.Spec.ClusterNetwork[0].HostPrefix != 64 {
			t.Errorf("Expected cluster network[0] hostPrefix 64, got %d", clusterInstance.Spec.ClusterNetwork[0].HostPrefix)
		}
	}

	// test-3node-siteconfig.yaml doesn't have machineNetwork defined
	if len(clusterInstance.Spec.MachineNetwork) != 0 {
		t.Errorf("Expected 0 machine networks, got %d", len(clusterInstance.Spec.MachineNetwork))
	}

	if len(clusterInstance.Spec.ServiceNetwork) != 1 {
		t.Errorf("Expected 1 service network, got %d", len(clusterInstance.Spec.ServiceNetwork))
	} else {
		if clusterInstance.Spec.ServiceNetwork[0].CIDR != "1001:2::/112" {
			t.Errorf("Expected service network[0] CIDR '1001:2::/112', got '%s'", clusterInstance.Spec.ServiceNetwork[0].CIDR)
		}
	}

	// Verify NTP Sources
	expectedNTPSources := []string{"1111:2222:3333:4444::2"}
	if len(clusterInstance.Spec.AdditionalNTPSources) != len(expectedNTPSources) {
		t.Errorf("Expected %d NTP sources, got %d", len(expectedNTPSources), len(clusterInstance.Spec.AdditionalNTPSources))
	}
	for i, expected := range expectedNTPSources {
		if i < len(clusterInstance.Spec.AdditionalNTPSources) && clusterInstance.Spec.AdditionalNTPSources[i] != expected {
			t.Errorf("Expected NTP source[%d] '%s', got '%s'", i, expected, clusterInstance.Spec.AdditionalNTPSources[i])
		}
	}

	// Verify Override Configs - test-3node-siteconfig.yaml doesn't have these
	if clusterInstance.Spec.InstallConfigOverrides != "" {
		t.Errorf("Expected installConfigOverrides to be empty, got '%s'", clusterInstance.Spec.InstallConfigOverrides)
	}
	if clusterInstance.Spec.IgnitionConfigOverride != "" {
		t.Errorf("Expected ignitionConfigOverride to be empty, got '%s'", clusterInstance.Spec.IgnitionConfigOverride)
	}

	// Verify Disk Encryption - test-3node-siteconfig.yaml doesn't have disk encryption
	if clusterInstance.Spec.DiskEncryption != nil {
		t.Error("Expected diskEncryption to be nil (not set in test-3node-siteconfig.yaml)")
	}

	// Verify Proxy - test-3node-siteconfig.yaml doesn't have proxy settings
	if clusterInstance.Spec.Proxy != nil {
		t.Error("Expected proxy to be nil (not set in test-3node-siteconfig.yaml)")
	}

	// Verify Platform and CPU - test-3node-siteconfig.yaml doesn't have these set
	if clusterInstance.Spec.PlatformType != "" {
		t.Errorf("Expected platformType to be empty, got '%s'", clusterInstance.Spec.PlatformType)
	}
	if clusterInstance.Spec.CPUArchitecture != "" {
		t.Errorf("Expected cpuArchitecture to be empty, got '%s'", clusterInstance.Spec.CPUArchitecture)
	}
	if clusterInstance.Spec.CPUPartitioningMode != "" {
		t.Errorf("Expected cpuPartitioningMode to be empty, got '%s'", clusterInstance.Spec.CPUPartitioningMode)
	}

	// Verify ExtraAnnotations - test-3node-siteconfig.yaml doesn't have CrAnnotations
	if clusterInstance.Spec.ExtraAnnotations != nil {
		t.Error("Expected extraAnnotations to be nil (not set in test-3node-siteconfig.yaml)")
	}

	// Verify ExtraLabels - test-3node-siteconfig.yaml has cluster labels
	if clusterInstance.Spec.ExtraLabels == nil {
		t.Error("Expected extraLabels to be set")
	} else {
		if managedClusterLabels, exists := clusterInstance.Spec.ExtraLabels["ManagedCluster"]; !exists {
			t.Error("Expected extraLabels.ManagedCluster to exist")
		} else {
			expectedLabels := map[string]string{
				"du-profile":     "latest",
				"common":         "true",
				"group-du-3node": "",
				"sites":          "example-multinode",
			}
			for key, expectedValue := range expectedLabels {
				if actualValue, exists := managedClusterLabels[key]; !exists {
					t.Errorf("Expected extraLabels.ManagedCluster['%s'] to exist", key)
				} else if actualValue != expectedValue {
					t.Errorf("Expected extraLabels.ManagedCluster['%s'] = '%s', got '%s'", key, expectedValue, actualValue)
				}
			}
		}
	}

	// Verify SuppressedManifests - test-3node-siteconfig.yaml doesn't have CrSuppression
	if len(clusterInstance.Spec.SuppressedManifests) != 0 {
		t.Errorf("Expected 0 suppressed manifests, got %d", len(clusterInstance.Spec.SuppressedManifests))
	}

	// Verify Template References
	if len(clusterInstance.Spec.TemplateRefs) != 1 {
		t.Errorf("Expected 1 template reference, got %d", len(clusterInstance.Spec.TemplateRefs))
	} else {
		if clusterInstance.Spec.TemplateRefs[0].Name != clusterTemplateName {
			t.Errorf("Expected template name '%s', got '%s'", clusterTemplateName, clusterInstance.Spec.TemplateRefs[0].Name)
		}
		if clusterInstance.Spec.TemplateRefs[0].Namespace != clusterTemplateNamespace {
			t.Errorf("Expected template namespace '%s', got '%s'", clusterTemplateNamespace, clusterInstance.Spec.TemplateRefs[0].Namespace)
		}
	}

	// Verify Nodes - test-3node-siteconfig.yaml has 3 master nodes
	if len(clusterInstance.Spec.Nodes) != 3 {
		t.Fatalf("Expected 3 nodes, got %d", len(clusterInstance.Spec.Nodes))
	}

	// Test first node in detail (has node labels)
	node1 := clusterInstance.Spec.Nodes[0]
	if node1.HostName != "example-node1.example.com" {
		t.Errorf("Expected node1 hostName 'example-node1.example.com', got '%s'", node1.HostName)
	}
	if node1.Role != "master" {
		t.Errorf("Expected node1 role 'master', got '%s'", node1.Role)
	}
	if node1.BmcAddress != "idrac-virtualmedia+https://[1111:2222:3333:4444::bbbb:1]/redfish/v1/Systems/System.Embedded.1" {
		t.Errorf("Expected node1 bmcAddress with IPv6, got '%s'", node1.BmcAddress)
	}
	if node1.BmcCredentialsName.Name != "example-node1-bmh-secret" {
		t.Errorf("Expected node1 bmcCredentialsName 'example-node1-bmh-secret', got '%s'", node1.BmcCredentialsName.Name)
	}
	if node1.BootMACAddress != "AA:BB:CC:DD:EE:11" {
		t.Errorf("Expected node1 bootMACAddress 'AA:BB:CC:DD:EE:11', got '%s'", node1.BootMACAddress)
	}
	if node1.BootMode != "UEFISecureBoot" {
		t.Errorf("Expected node1 bootMode 'UEFISecureBoot', got '%s'", node1.BootMode)
	}

	// Verify Node1 Root Device Hints
	if node1.RootDeviceHints == nil {
		t.Error("Expected node1 rootDeviceHints to be set")
	} else {
		if deviceName, exists := node1.RootDeviceHints["deviceName"]; !exists {
			t.Error("Expected node1 rootDeviceHints.deviceName to exist")
		} else if deviceName != "/dev/disk/by-path/pci-0000:01:00.0-scsi-0:2:0:0" {
			t.Errorf("Expected node1 rootDeviceHints.deviceName '/dev/disk/by-path/pci-0000:01:00.0-scsi-0:2:0:0', got '%v'", deviceName)
		}
	}

	// Verify Node1 Labels (should NOT have extraLabels.node)
	if node1.NodeLabels == nil {
		t.Error("Expected node1 nodeLabels to be set")
	} else {
		expectedNodeLabels := map[string]string{
			"node-role.kubernetes.io/master-du": "",
			"custom-label/parameter1":           "true",
		}
		for key, expectedValue := range expectedNodeLabels {
			if actualValue, exists := node1.NodeLabels[key]; !exists {
				t.Errorf("Expected node1 nodeLabels['%s'] to exist", key)
			} else if actualValue != expectedValue {
				t.Errorf("Expected node1 nodeLabels['%s'] = '%s', got '%s'", key, expectedValue, actualValue)
			}
		}
	}

	// Verify Node1 ExtraLabels should NOT be set (this was the fix)
	if node1.ExtraLabels != nil {
		t.Error("Expected node1 extraLabels to be nil (should not duplicate nodeLabels)")
	}

	// Verify Node1 Network
	if node1.NodeNetwork == nil {
		t.Error("Expected node1 nodeNetwork to be set")
	} else {
		if node1.NodeNetwork.Config == nil {
			t.Error("Expected node1 nodeNetwork.config to be set")
		}
		if len(node1.NodeNetwork.Interfaces) != 1 {
			t.Errorf("Expected 1 node1 network interface, got %d", len(node1.NodeNetwork.Interfaces))
		} else {
			if node1.NodeNetwork.Interfaces[0].Name != "eno1" {
				t.Errorf("Expected node1 network interface name 'eno1', got '%s'", node1.NodeNetwork.Interfaces[0].Name)
			}
			if node1.NodeNetwork.Interfaces[0].MacAddress != "AA:BB:CC:DD:EE:11" {
				t.Errorf("Expected node1 network interface MAC 'AA:BB:CC:DD:EE:11', got '%s'", node1.NodeNetwork.Interfaces[0].MacAddress)
			}
		}
	}

	// Verify Node1 Template References
	if len(node1.TemplateRefs) != 1 {
		t.Errorf("Expected 1 node1 template reference, got %d", len(node1.TemplateRefs))
	} else {
		if node1.TemplateRefs[0].Name != nodeTemplateName {
			t.Errorf("Expected node1 template name '%s', got '%s'", nodeTemplateName, node1.TemplateRefs[0].Name)
		}
		if node1.TemplateRefs[0].Namespace != nodeTemplateNamespace {
			t.Errorf("Expected node1 template namespace '%s', got '%s'", nodeTemplateNamespace, node1.TemplateRefs[0].Namespace)
		}
	}

	// Test second and third nodes for basic consistency
	expectedHostNames := []string{"example-node1.example.com", "example-node2.example.com", "example-node3.example.com"}
	expectedBMCAddresses := []string{
		"idrac-virtualmedia+https://[1111:2222:3333:4444::bbbb:1]/redfish/v1/Systems/System.Embedded.1",
		"idrac-virtualmedia+https://[1111:2222:3333:4444::bbbb:2]/redfish/v1/Systems/System.Embedded.1",
		"idrac-virtualmedia+https://[1111:2222:3333:4444::bbbb:3]/redfish/v1/Systems/System.Embedded.1",
	}
	expectedMACs := []string{"AA:BB:CC:DD:EE:11", "AA:BB:CC:DD:EE:22", "AA:BB:CC:DD:EE:33"}
	expectedCredentials := []string{"example-node1-bmh-secret", "example-node2-bmh-secret", "example-node3-bmh-secret"}

	for i, node := range clusterInstance.Spec.Nodes {
		if node.HostName != expectedHostNames[i] {
			t.Errorf("Expected node[%d] hostName '%s', got '%s'", i, expectedHostNames[i], node.HostName)
		}
		if node.BmcAddress != expectedBMCAddresses[i] {
			t.Errorf("Expected node[%d] bmcAddress '%s', got '%s'", i, expectedBMCAddresses[i], node.BmcAddress)
		}
		if node.BootMACAddress != expectedMACs[i] {
			t.Errorf("Expected node[%d] bootMACAddress '%s', got '%s'", i, expectedMACs[i], node.BootMACAddress)
		}
		if node.BmcCredentialsName.Name != expectedCredentials[i] {
			t.Errorf("Expected node[%d] bmcCredentialsName '%s', got '%s'", i, expectedCredentials[i], node.BmcCredentialsName.Name)
		}
		if node.Role != "master" {
			t.Errorf("Expected node[%d] role 'master', got '%s'", i, node.Role)
		}
		if node.BootMode != "UEFISecureBoot" {
			t.Errorf("Expected node[%d] bootMode 'UEFISecureBoot', got '%s'", i, node.BootMode)
		}

		// Verify all nodes have NodeNetwork configured
		if node.NodeNetwork == nil {
			t.Errorf("Expected node[%d] nodeNetwork to be set", i)
		}

		// Verify nodes 2 and 3 don't have node labels (only node 1 does)
		if i > 0 && len(node.NodeLabels) > 0 {
			t.Errorf("Expected node[%d] to have no nodeLabels, got %v", i, node.NodeLabels)
		}

		// Verify all nodes have extraLabels nil
		if node.ExtraLabels != nil {
			t.Errorf("Expected node[%d] extraLabels to be nil", i)
		}
	}

	// Verify default extraManifestsRefs (should contain cluster name when no manifests provided)
	if len(clusterInstance.Spec.ExtraManifestsRefs) != 1 {
		t.Errorf("Expected extraManifestsRefs to contain 1 default entry, got %d", len(clusterInstance.Spec.ExtraManifestsRefs))
	} else if clusterInstance.Spec.ExtraManifestsRefs[0].Name != "example-3node" {
		t.Errorf("Expected default extraManifestsRefs[0].Name to be 'example-3node', got '%s'", clusterInstance.Spec.ExtraManifestsRefs[0].Name)
	}
	if clusterInstance.Spec.CaBundleRef != nil {
		t.Error("Expected caBundleRef to be nil")
	}
	if clusterInstance.Spec.Reinstall != nil {
		t.Error("Expected reinstall to be nil")
	}
	if len(clusterInstance.Spec.PruneManifests) > 0 {
		t.Error("Expected pruneManifests to be empty")
	}
}

func TestComprehensive5NodeFieldConversion(t *testing.T) {
	// Read the actual test-5node-siteconfig.yaml file
	siteConfig, err := readSiteConfig("samples/test-5node-siteconfig.yaml")
	if err != nil {
		t.Fatalf("Failed to read test-5node-siteconfig.yaml: %v", err)
	}

	// Validate that it's a SiteConfig
	if siteConfig.Kind != "SiteConfig" {
		t.Fatalf("Expected Kind to be 'SiteConfig', got '%s'", siteConfig.Kind)
	}

	if len(siteConfig.Spec.Clusters) != 1 {
		t.Fatalf("Expected 1 cluster in test-5node-siteconfig.yaml, got %d", len(siteConfig.Spec.Clusters))
	}

	// Convert to ClusterInstance
	cluster := siteConfig.Spec.Clusters[0]
	clusterTemplateNamespace := "test-cluster-namespace"
	clusterTemplateName := "test-cluster-template"
	nodeTemplateNamespace := "test-node-namespace"
	nodeTemplateName := "test-node-template"

	clusterInstance := convertClusterToClusterInstance(siteConfig, cluster, createTemplateRefs(clusterTemplateNamespace, clusterTemplateName), createTemplateRefs(nodeTemplateNamespace, nodeTemplateName), []LocalObjectReference{}, "", &WarningsCollector{}, 0, "test-siteconfig.yaml")

	// Verify API Version and Kind
	if clusterInstance.ApiVersion != "siteconfig.open-cluster-management.io/v1alpha1" {
		t.Errorf("Expected apiVersion 'siteconfig.open-cluster-management.io/v1alpha1', got '%s'", clusterInstance.ApiVersion)
	}
	if clusterInstance.Kind != "ClusterInstance" {
		t.Errorf("Expected kind 'ClusterInstance', got '%s'", clusterInstance.Kind)
	}

	// Verify Metadata
	if clusterInstance.Metadata.Name != "example-standard" {
		t.Errorf("Expected metadata name 'example-standard', got '%s'", clusterInstance.Metadata.Name)
	}
	if clusterInstance.Metadata.Namespace != "example-standard" {
		t.Errorf("Expected metadata namespace 'example-standard', got '%s'", clusterInstance.Metadata.Namespace)
	}

	// Verify Spec - Basic Fields
	if clusterInstance.Spec.BaseDomain != "example.com" {
		t.Errorf("Expected baseDomain 'example.com', got '%s'", clusterInstance.Spec.BaseDomain)
	}
	if clusterInstance.Spec.PullSecretRef.Name != "assisted-deployment-pull-secret" {
		t.Errorf("Expected pullSecretRef name 'assisted-deployment-pull-secret', got '%s'", clusterInstance.Spec.PullSecretRef.Name)
	}
	if clusterInstance.Spec.ClusterImageSetNameRef != "openshift-4.19" {
		t.Errorf("Expected clusterImageSetNameRef 'openshift-4.19', got '%s'", clusterInstance.Spec.ClusterImageSetNameRef)
	}
	if clusterInstance.Spec.SshPublicKey != "ssh-rsa AAAA..." {
		t.Errorf("Expected sshPublicKey 'ssh-rsa AAAA...', got '%s'", clusterInstance.Spec.SshPublicKey)
	}
	if clusterInstance.Spec.ClusterName != "example-standard" {
		t.Errorf("Expected clusterName 'example-standard', got '%s'", clusterInstance.Spec.ClusterName)
	}
	if clusterInstance.Spec.ClusterType != "HighlyAvailable" {
		t.Errorf("Expected clusterType 'HighlyAvailable' for 5-node cluster, got '%s'", clusterInstance.Spec.ClusterType)
	}
	if clusterInstance.Spec.NetworkType != "OVNKubernetes" {
		t.Errorf("Expected networkType 'OVNKubernetes', got '%s'", clusterInstance.Spec.NetworkType)
	}

	// Verify VIPs - test-5node-siteconfig.yaml has same IPv6 VIPs as 3-node
	expectedAPIVIPs := []string{"1111:2222:3333:4444::1:1"}
	if len(clusterInstance.Spec.ApiVIPs) != len(expectedAPIVIPs) {
		t.Errorf("Expected %d API VIPs, got %d", len(expectedAPIVIPs), len(clusterInstance.Spec.ApiVIPs))
	}
	for i, expected := range expectedAPIVIPs {
		if i < len(clusterInstance.Spec.ApiVIPs) && clusterInstance.Spec.ApiVIPs[i] != expected {
			t.Errorf("Expected API VIP[%d] '%s', got '%s'", i, expected, clusterInstance.Spec.ApiVIPs[i])
		}
	}

	expectedIngressVIPs := []string{"1111:2222:3333:4444::1:2"}
	if len(clusterInstance.Spec.IngressVIPs) != len(expectedIngressVIPs) {
		t.Errorf("Expected %d Ingress VIPs, got %d", len(expectedIngressVIPs), len(clusterInstance.Spec.IngressVIPs))
	}
	for i, expected := range expectedIngressVIPs {
		if i < len(clusterInstance.Spec.IngressVIPs) && clusterInstance.Spec.IngressVIPs[i] != expected {
			t.Errorf("Expected Ingress VIP[%d] '%s', got '%s'", i, expected, clusterInstance.Spec.IngressVIPs[i])
		}
	}

	// Verify HoldInstallation - should be false by default
	if clusterInstance.Spec.HoldInstallation != false {
		t.Errorf("Expected holdInstallation false, got %t", clusterInstance.Spec.HoldInstallation)
	}

	// Verify Networks - test-5node-siteconfig.yaml has same IPv6 networks as 3-node
	if len(clusterInstance.Spec.ClusterNetwork) != 1 {
		t.Errorf("Expected 1 cluster network, got %d", len(clusterInstance.Spec.ClusterNetwork))
	} else {
		if clusterInstance.Spec.ClusterNetwork[0].CIDR != "1001:1::/48" {
			t.Errorf("Expected cluster network[0] CIDR '1001:1::/48', got '%s'", clusterInstance.Spec.ClusterNetwork[0].CIDR)
		}
		if clusterInstance.Spec.ClusterNetwork[0].HostPrefix != 64 {
			t.Errorf("Expected cluster network[0] hostPrefix 64, got %d", clusterInstance.Spec.ClusterNetwork[0].HostPrefix)
		}
	}

	// test-5node-siteconfig.yaml doesn't have machineNetwork defined
	if len(clusterInstance.Spec.MachineNetwork) != 0 {
		t.Errorf("Expected 0 machine networks, got %d", len(clusterInstance.Spec.MachineNetwork))
	}

	if len(clusterInstance.Spec.ServiceNetwork) != 1 {
		t.Errorf("Expected 1 service network, got %d", len(clusterInstance.Spec.ServiceNetwork))
	} else {
		if clusterInstance.Spec.ServiceNetwork[0].CIDR != "1001:2::/112" {
			t.Errorf("Expected service network[0] CIDR '1001:2::/112', got '%s'", clusterInstance.Spec.ServiceNetwork[0].CIDR)
		}
	}

	// Verify NTP Sources
	expectedNTPSources := []string{"1111:2222:3333:4444::2"}
	if len(clusterInstance.Spec.AdditionalNTPSources) != len(expectedNTPSources) {
		t.Errorf("Expected %d NTP sources, got %d", len(expectedNTPSources), len(clusterInstance.Spec.AdditionalNTPSources))
	}
	for i, expected := range expectedNTPSources {
		if i < len(clusterInstance.Spec.AdditionalNTPSources) && clusterInstance.Spec.AdditionalNTPSources[i] != expected {
			t.Errorf("Expected NTP source[%d] '%s', got '%s'", i, expected, clusterInstance.Spec.AdditionalNTPSources[i])
		}
	}

	// Verify Override Configs - test-5node-siteconfig.yaml doesn't have these
	if clusterInstance.Spec.InstallConfigOverrides != "" {
		t.Errorf("Expected installConfigOverrides to be empty, got '%s'", clusterInstance.Spec.InstallConfigOverrides)
	}
	if clusterInstance.Spec.IgnitionConfigOverride != "" {
		t.Errorf("Expected ignitionConfigOverride to be empty, got '%s'", clusterInstance.Spec.IgnitionConfigOverride)
	}

	// Verify Disk Encryption - test-5node-siteconfig.yaml doesn't have disk encryption
	if clusterInstance.Spec.DiskEncryption != nil {
		t.Error("Expected diskEncryption to be nil (not set in test-5node-siteconfig.yaml)")
	}

	// Verify Proxy - test-5node-siteconfig.yaml doesn't have proxy settings
	if clusterInstance.Spec.Proxy != nil {
		t.Error("Expected proxy to be nil (not set in test-5node-siteconfig.yaml)")
	}

	// Verify Platform and CPU - test-5node-siteconfig.yaml doesn't have these set
	if clusterInstance.Spec.PlatformType != "" {
		t.Errorf("Expected platformType to be empty, got '%s'", clusterInstance.Spec.PlatformType)
	}
	if clusterInstance.Spec.CPUArchitecture != "" {
		t.Errorf("Expected cpuArchitecture to be empty, got '%s'", clusterInstance.Spec.CPUArchitecture)
	}
	if clusterInstance.Spec.CPUPartitioningMode != "" {
		t.Errorf("Expected cpuPartitioningMode to be empty, got '%s'", clusterInstance.Spec.CPUPartitioningMode)
	}

	// Verify ExtraAnnotations - test-5node-siteconfig.yaml doesn't have CrAnnotations
	if clusterInstance.Spec.ExtraAnnotations != nil {
		t.Error("Expected extraAnnotations to be nil (not set in test-5node-siteconfig.yaml)")
	}

	// Verify ExtraLabels - test-5node-siteconfig.yaml has cluster labels
	if clusterInstance.Spec.ExtraLabels == nil {
		t.Error("Expected extraLabels to be set")
	} else {
		if managedClusterLabels, exists := clusterInstance.Spec.ExtraLabels["ManagedCluster"]; !exists {
			t.Error("Expected extraLabels.ManagedCluster to exist")
		} else {
			expectedLabels := map[string]string{
				"du-profile":        "latest",
				"common":            "true",
				"group-du-standard": "",
				"sites":             "example-multinode",
			}
			for key, expectedValue := range expectedLabels {
				if actualValue, exists := managedClusterLabels[key]; !exists {
					t.Errorf("Expected extraLabels.ManagedCluster['%s'] to exist", key)
				} else if actualValue != expectedValue {
					t.Errorf("Expected extraLabels.ManagedCluster['%s'] = '%s', got '%s'", key, expectedValue, actualValue)
				}
			}
		}
	}

	// Verify SuppressedManifests - test-5node-siteconfig.yaml doesn't have CrSuppression
	if len(clusterInstance.Spec.SuppressedManifests) != 0 {
		t.Errorf("Expected 0 suppressed manifests, got %d", len(clusterInstance.Spec.SuppressedManifests))
	}

	// Verify Template References
	if len(clusterInstance.Spec.TemplateRefs) != 1 {
		t.Errorf("Expected 1 template reference, got %d", len(clusterInstance.Spec.TemplateRefs))
	} else {
		if clusterInstance.Spec.TemplateRefs[0].Name != clusterTemplateName {
			t.Errorf("Expected template name '%s', got '%s'", clusterTemplateName, clusterInstance.Spec.TemplateRefs[0].Name)
		}
		if clusterInstance.Spec.TemplateRefs[0].Namespace != clusterTemplateNamespace {
			t.Errorf("Expected template namespace '%s', got '%s'", clusterTemplateNamespace, clusterInstance.Spec.TemplateRefs[0].Namespace)
		}
	}

	// Verify Nodes - test-5node-siteconfig.yaml has 5 nodes (3 masters + 2 workers)
	if len(clusterInstance.Spec.Nodes) != 5 {
		t.Fatalf("Expected 5 nodes, got %d", len(clusterInstance.Spec.Nodes))
	}

	// Define expected values for all 5 nodes
	expectedHostNames := []string{
		"example-node1.example.com", "example-node2.example.com", "example-node3.example.com",
		"example-node4.example.com", "example-node5.example.com",
	}
	expectedBMCAddresses := []string{
		"idrac-virtualmedia+https://[1111:2222:3333:4444::bbbb:1]/redfish/v1/Systems/System.Embedded.1",
		"idrac-virtualmedia+https://[1111:2222:3333:4444::bbbb:2]/redfish/v1/Systems/System.Embedded.1",
		"idrac-virtualmedia+https://[1111:2222:3333:4444::bbbb:3]/redfish/v1/Systems/System.Embedded.1",
		"idrac-virtualmedia+https://[1111:2222:3333:4444::bbbb:4]/redfish/v1/Systems/System.Embedded.1",
		"idrac-virtualmedia+https://[1111:2222:3333:4444::bbbb:5]/redfish/v1/Systems/System.Embedded.1",
	}
	expectedMACs := []string{"AA:BB:CC:DD:EE:11", "AA:BB:CC:DD:EE:22", "AA:BB:CC:DD:EE:33", "AA:BB:CC:DD:EE:44", "AA:BB:CC:DD:EE:55"}
	expectedCredentials := []string{
		"example-node1-bmh-secret", "example-node2-bmh-secret", "example-node3-bmh-secret",
		"example-node4-bmh-secret", "example-node5-bmh-secret",
	}
	expectedRoles := []string{"master", "master", "master", "worker", "worker"}

	// Test all nodes for consistency
	for i, node := range clusterInstance.Spec.Nodes {
		if node.HostName != expectedHostNames[i] {
			t.Errorf("Expected node[%d] hostName '%s', got '%s'", i, expectedHostNames[i], node.HostName)
		}
		if node.BmcAddress != expectedBMCAddresses[i] {
			t.Errorf("Expected node[%d] bmcAddress '%s', got '%s'", i, expectedBMCAddresses[i], node.BmcAddress)
		}
		if node.BootMACAddress != expectedMACs[i] {
			t.Errorf("Expected node[%d] bootMACAddress '%s', got '%s'", i, expectedMACs[i], node.BootMACAddress)
		}
		if node.BmcCredentialsName.Name != expectedCredentials[i] {
			t.Errorf("Expected node[%d] bmcCredentialsName '%s', got '%s'", i, expectedCredentials[i], node.BmcCredentialsName.Name)
		}
		if node.Role != expectedRoles[i] {
			t.Errorf("Expected node[%d] role '%s', got '%s'", i, expectedRoles[i], node.Role)
		}
		if node.BootMode != "UEFISecureBoot" {
			t.Errorf("Expected node[%d] bootMode 'UEFISecureBoot', got '%s'", i, node.BootMode)
		}

		// Verify root device hints
		if node.RootDeviceHints == nil {
			t.Errorf("Expected node[%d] rootDeviceHints to be set", i)
		} else {
			if deviceName, exists := node.RootDeviceHints["deviceName"]; !exists {
				t.Errorf("Expected node[%d] rootDeviceHints.deviceName to exist", i)
			} else if deviceName != "/dev/disk/by-path/pci-0000:01:00.0-scsi-0:2:0:0" {
				t.Errorf("Expected node[%d] rootDeviceHints.deviceName '/dev/disk/by-path/pci-0000:01:00.0-scsi-0:2:0:0', got '%v'", i, deviceName)
			}
		}

		// Verify all nodes have NodeNetwork configured
		if node.NodeNetwork == nil {
			t.Errorf("Expected node[%d] nodeNetwork to be set", i)
		} else {
			if node.NodeNetwork.Config == nil {
				t.Errorf("Expected node[%d] nodeNetwork.config to be set", i)
			}
			if len(node.NodeNetwork.Interfaces) != 1 {
				t.Errorf("Expected 1 network interface for node[%d], got %d", i, len(node.NodeNetwork.Interfaces))
			} else {
				if node.NodeNetwork.Interfaces[0].Name != "eno1" {
					t.Errorf("Expected node[%d] network interface name 'eno1', got '%s'", i, node.NodeNetwork.Interfaces[0].Name)
				}
				if node.NodeNetwork.Interfaces[0].MacAddress != expectedMACs[i] {
					t.Errorf("Expected node[%d] network interface MAC '%s', got '%s'", i, expectedMACs[i], node.NodeNetwork.Interfaces[0].MacAddress)
				}
			}
		}

		// Verify node labels - test-5node-siteconfig.yaml doesn't have any node labels
		if len(node.NodeLabels) > 0 {
			t.Errorf("Expected node[%d] to have no nodeLabels, got %v", i, node.NodeLabels)
		}

		// Verify all nodes have extraLabels nil (the fix we implemented)
		if node.ExtraLabels != nil {
			t.Errorf("Expected node[%d] extraLabels to be nil", i)
		}

		// Verify template references
		if len(node.TemplateRefs) != 1 {
			t.Errorf("Expected 1 template reference for node[%d], got %d", i, len(node.TemplateRefs))
		} else {
			if node.TemplateRefs[0].Name != nodeTemplateName {
				t.Errorf("Expected node[%d] template name '%s', got '%s'", i, nodeTemplateName, node.TemplateRefs[0].Name)
			}
			if node.TemplateRefs[0].Namespace != nodeTemplateNamespace {
				t.Errorf("Expected node[%d] template namespace '%s', got '%s'", i, nodeTemplateNamespace, node.TemplateRefs[0].Namespace)
			}
		}

		// Verify node-level fields that should be empty
		if node.AutomatedCleaningMode != "" {
			t.Errorf("Expected node[%d] automatedCleaningMode to be empty, got '%s'", i, node.AutomatedCleaningMode)
		}
		if node.InstallerArgs != "" {
			t.Errorf("Expected node[%d] installerArgs to be empty, got '%s'", i, node.InstallerArgs)
		}
		if node.IronicInspect != "" {
			t.Errorf("Expected node[%d] ironicInspect to be empty, got '%s'", i, node.IronicInspect)
		}
		if node.IgnitionConfigOverride != "" {
			t.Errorf("Expected node[%d] ignitionConfigOverride to be empty, got '%s'", i, node.IgnitionConfigOverride)
		}
		if node.ExtraAnnotations != nil {
			t.Errorf("Expected node[%d] extraAnnotations to be nil", i)
		}
		if len(node.SuppressedManifests) > 0 {
			t.Errorf("Expected node[%d] suppressedManifests to be empty, got %v", i, node.SuppressedManifests)
		}
		if len(node.PruneManifests) > 0 {
			t.Errorf("Expected node[%d] pruneManifests to be empty, got %v", i, node.PruneManifests)
		}
		if node.HostRef != nil {
			t.Errorf("Expected node[%d] hostRef to be nil", i)
		}
		if node.CPUArchitecture != "" {
			t.Errorf("Expected node[%d] cpuArchitecture to be empty, got '%s'", i, node.CPUArchitecture)
		}
	}

	// Verify master vs worker node distribution
	masterCount := 0
	workerCount := 0
	for _, node := range clusterInstance.Spec.Nodes {
		switch node.Role {
		case "master":
			masterCount++
		case "worker":
			workerCount++
		}
	}

	if masterCount != 3 {
		t.Errorf("Expected 3 master nodes, got %d", masterCount)
	}
	if workerCount != 2 {
		t.Errorf("Expected 2 worker nodes, got %d", workerCount)
	}

	// Verify default extraManifestsRefs (should contain cluster name when no manifests provided)
	if len(clusterInstance.Spec.ExtraManifestsRefs) != 1 {
		t.Errorf("Expected extraManifestsRefs to contain 1 default entry, got %d", len(clusterInstance.Spec.ExtraManifestsRefs))
	} else if clusterInstance.Spec.ExtraManifestsRefs[0].Name != "example-standard" {
		t.Errorf("Expected default extraManifestsRefs[0].Name to be 'example-standard', got '%s'", clusterInstance.Spec.ExtraManifestsRefs[0].Name)
	}
	if clusterInstance.Spec.CaBundleRef != nil {
		t.Error("Expected caBundleRef to be nil")
	}
	if clusterInstance.Spec.Reinstall != nil {
		t.Error("Expected reinstall to be nil")
	}
	if len(clusterInstance.Spec.PruneManifests) > 0 {
		t.Error("Expected pruneManifests to be empty")
	}
}

func TestExtraManifestsRefsMerging(t *testing.T) {
	// Test cases to verify merging of manifestsConfigMapRefs from SiteConfig with extraManifestsRefs from command line
	testCases := []struct {
		name                    string
		siteConfigManifestsRefs []ManifestsConfigMapReference
		cmdLineManifestsRefs    string
		expectedResult          []string
		description             string
	}{
		{
			name:                    "Both SiteConfig and command line have manifests",
			siteConfigManifestsRefs: []ManifestsConfigMapReference{{Name: "siteconfig-cm1"}, {Name: "siteconfig-cm2"}},
			cmdLineManifestsRefs:    "cmdline-cm1,cmdline-cm2",
			expectedResult:          []string{"siteconfig-cm1", "siteconfig-cm2", "cmdline-cm1", "cmdline-cm2"},
			description:             "Should merge both SiteConfig and command line manifests",
		},
		{
			name:                    "Only SiteConfig has manifests",
			siteConfigManifestsRefs: []ManifestsConfigMapReference{{Name: "siteconfig-only1"}, {Name: "siteconfig-only2"}},
			cmdLineManifestsRefs:    "",
			expectedResult:          []string{"siteconfig-only1", "siteconfig-only2"},
			description:             "Should use only SiteConfig manifests when command line is empty",
		},
		{
			name:                    "Only command line has manifests",
			siteConfigManifestsRefs: []ManifestsConfigMapReference{},
			cmdLineManifestsRefs:    "cmdline-only1,cmdline-only2",
			expectedResult:          []string{"cmdline-only1", "cmdline-only2"},
			description:             "Should use only command line manifests when SiteConfig is empty",
		},
		{
			name:                    "Neither has manifests",
			siteConfigManifestsRefs: []ManifestsConfigMapReference{},
			cmdLineManifestsRefs:    "",
			expectedResult:          []string{"test-cluster"},
			description:             "Should use cluster name as default when both are empty",
		},
		{
			name:                    "Command line with whitespace",
			siteConfigManifestsRefs: []ManifestsConfigMapReference{{Name: "siteconfig-cm1"}},
			cmdLineManifestsRefs:    " cmdline-cm1 , cmdline-cm2 , cmdline-cm3 ",
			expectedResult:          []string{"siteconfig-cm1", "cmdline-cm1", "cmdline-cm2", "cmdline-cm3"},
			description:             "Should handle whitespace in command line arguments",
		},
		{
			name:                    "Command line with empty entries",
			siteConfigManifestsRefs: []ManifestsConfigMapReference{{Name: "siteconfig-cm1"}},
			cmdLineManifestsRefs:    "cmdline-cm1,,cmdline-cm2,",
			expectedResult:          []string{"siteconfig-cm1", "cmdline-cm1", "cmdline-cm2"},
			description:             "Should skip empty entries in command line arguments",
		},
		{
			name:                    "Duplicate names between SiteConfig and command line",
			siteConfigManifestsRefs: []ManifestsConfigMapReference{{Name: "common-cm"}, {Name: "siteconfig-cm"}},
			cmdLineManifestsRefs:    "common-cm,cmdline-cm",
			expectedResult:          []string{"common-cm", "siteconfig-cm", "common-cm", "cmdline-cm"},
			description:             "Should include duplicates (no deduplication)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test SiteConfig with the specified manifestsConfigMapRefs
			siteConfig := &SiteConfig{
				ApiVersion: "ran.openshift.io/v1",
				Kind:       "SiteConfig",
				Metadata: Metadata{
					Name:      "test-site",
					Namespace: "test-namespace",
				},
				Spec: Spec{
					BaseDomain: "example.com",
					PullSecretRef: PullSecretRef{
						Name: "pull-secret",
					},
					ClusterImageSetNameRef: "img-set",
					SshPublicKey:           "ssh-rsa test-key",
					Clusters: []Cluster{
						{
							ClusterName:            "test-cluster",
							ManifestsConfigMapRefs: tc.siteConfigManifestsRefs,
							Nodes: []Node{
								{
									HostName:   "node1.example.com",
									BmcAddress: "redfish://192.168.1.1/redfish/v1/Systems/1",
									BmcCredentialsName: BmcCredentialsName{
										Name: "node1-secret",
									},
									BootMACAddress: "AA:BB:CC:DD:EE:FF",
									Role:           "master",
								},
							},
						},
					},
				},
			}

			cluster := siteConfig.Spec.Clusters[0]

			// Parse command line manifests refs (similar to convertToClusterInstance function)
			var cmdLineManifestsRefs []LocalObjectReference
			if tc.cmdLineManifestsRefs != "" {
				manifestNames := strings.Split(tc.cmdLineManifestsRefs, ",")
				for _, name := range manifestNames {
					name = strings.TrimSpace(name)
					if name != "" {
						cmdLineManifestsRefs = append(cmdLineManifestsRefs, LocalObjectReference{Name: name})
					}
				}
			}

			// Test the conversion
			clusterTemplateNamespace := "test-namespace"
			clusterTemplateName := "test-template"
			nodeTemplateNamespace := "test-namespace"
			nodeTemplateName := "test-node-template"

			clusterInstance := convertClusterToClusterInstance(
				siteConfig,
				cluster,
				createTemplateRefs(clusterTemplateNamespace, clusterTemplateName),
				createTemplateRefs(nodeTemplateNamespace, nodeTemplateName),
				cmdLineManifestsRefs,
				"", // No suppressed manifests for this test
				&WarningsCollector{},
				0,
				"test-siteconfig.yaml",
			)

			// Verify the merged extraManifestsRefs
			actualManifests := make([]string, len(clusterInstance.Spec.ExtraManifestsRefs))
			for i, ref := range clusterInstance.Spec.ExtraManifestsRefs {
				actualManifests[i] = ref.Name
			}

			if len(actualManifests) != len(tc.expectedResult) {
				t.Errorf("Test %s: Expected %d manifests, got %d. Expected: %v, Got: %v",
					tc.name, len(tc.expectedResult), len(actualManifests), tc.expectedResult, actualManifests)
				return
			}

			for i, expected := range tc.expectedResult {
				if i >= len(actualManifests) || actualManifests[i] != expected {
					t.Errorf("Test %s: At index %d, expected '%s', got '%s'. Full expected: %v, Full actual: %v",
						tc.name, i, expected, actualManifests[i], tc.expectedResult, actualManifests)
					break
				}
			}

			t.Logf("Test %s passed: %s", tc.name, tc.description)
		})
	}
}

func TestExtraManifestsRefsWithRealSiteConfig(t *testing.T) {
	// Create a SiteConfig YAML string with manifestsConfigMapRefs
	siteConfigYAML := `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: test-manifests-site
  namespace: test-namespace
spec:
  baseDomain: example.com
  pullSecretRef:
    name: pull-secret
  clusterImageSetNameRef: img-set
  sshPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQD"
  clusters:
  - clusterName: test-cluster
    manifestsConfigMapRefs:
      - name: siteconfig-manifest1
      - name: siteconfig-manifest2
    nodes:
    - hostName: node1.example.com
      bmcAddress: redfish://192.168.1.1/redfish/v1/Systems/1
      bmcCredentialsName:
        name: node1-secret
      bootMACAddress: AA:BB:CC:DD:EE:FF
      role: master
`

	// Write to temp file
	tempFile := "test-manifests-refs.yaml"
	err := os.WriteFile(tempFile, []byte(siteConfigYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp SiteConfig file: %v", err)
	}
	defer os.Remove(tempFile)

	// Read the SiteConfig
	siteConfig, err := readSiteConfig(tempFile)
	if err != nil {
		t.Fatalf("Failed to read SiteConfig: %v", err)
	}

	cluster := siteConfig.Spec.Clusters[0]

	// Test with command line manifests
	cmdLineManifestsRefs := []LocalObjectReference{
		{Name: "cmdline-cm1"},
		{Name: "cmdline-cm2"},
	}

	clusterInstance := convertClusterToClusterInstance(
		siteConfig,
		cluster,
		createTemplateRefs("test-namespace", "test-template"),
		createTemplateRefs("test-namespace", "test-node-template"),
		cmdLineManifestsRefs,
		"",
		&WarningsCollector{},
		0,
		"test-siteconfig.yaml",
	)

	// Verify the results
	expectedManifests := []string{"siteconfig-manifest1", "siteconfig-manifest2", "cmdline-cm1", "cmdline-cm2"}
	actualManifests := make([]string, len(clusterInstance.Spec.ExtraManifestsRefs))
	for i, ref := range clusterInstance.Spec.ExtraManifestsRefs {
		actualManifests[i] = ref.Name
	}

	if len(actualManifests) != len(expectedManifests) {
		t.Errorf("Expected %d manifests, got %d. Expected: %v, Got: %v",
			len(expectedManifests), len(actualManifests), expectedManifests, actualManifests)
		return
	}

	for i, expected := range expectedManifests {
		if actualManifests[i] != expected {
			t.Errorf("At index %d, expected '%s', got '%s'", i, expected, actualManifests[i])
		}
	}

	// Verify SiteConfig manifestsConfigMapRefs were properly read
	if len(cluster.ManifestsConfigMapRefs) != 2 {
		t.Errorf("Expected 2 manifestsConfigMapRefs in SiteConfig, got %d", len(cluster.ManifestsConfigMapRefs))
	}

	expectedSiteConfigRefs := []string{"siteconfig-manifest1", "siteconfig-manifest2"}
	for i, expected := range expectedSiteConfigRefs {
		if i >= len(cluster.ManifestsConfigMapRefs) || cluster.ManifestsConfigMapRefs[i].Name != expected {
			t.Errorf("Expected manifestsConfigMapRefs[%d] to be '%s', got '%s'",
				i, expected, cluster.ManifestsConfigMapRefs[i].Name)
		}
	}

	t.Log("Successfully verified merging of SiteConfig manifestsConfigMapRefs with command line extraManifestsRefs")
}

func TestComprehensiveSampleConversion(t *testing.T) {
	// Test comprehensive field conversion using the actual samples/comprehensive-siteconfig.yaml file
	// This verifies that our comprehensive sample works correctly and all supported fields are converted

	// Read the comprehensive sample SiteConfig
	siteConfig, err := readSiteConfig("samples/comprehensive-siteconfig.yaml")
	if err != nil {
		t.Fatalf("Failed to read comprehensive SiteConfig: %v", err)
	}

	// Verify the SiteConfig was read correctly
	if siteConfig.Metadata.Name != "comprehensive-example" {
		t.Errorf("Expected metadata name 'comprehensive-example', got '%s'", siteConfig.Metadata.Name)
	}
	if siteConfig.Metadata.Namespace != "comprehensive-example" {
		t.Errorf("Expected metadata namespace 'comprehensive-example', got '%s'", siteConfig.Metadata.Namespace)
	}

	// Verify basic spec fields
	if siteConfig.Spec.BaseDomain != "example.com" {
		t.Errorf("Expected baseDomain 'example.com', got '%s'", siteConfig.Spec.BaseDomain)
	}
	if siteConfig.Spec.PullSecretRef.Name != "pull-secret" {
		t.Errorf("Expected pullSecretRef name 'pull-secret', got '%s'", siteConfig.Spec.PullSecretRef.Name)
	}
	if siteConfig.Spec.ClusterImageSetNameRef != "openshift-4.19" {
		t.Errorf("Expected clusterImageSetNameRef 'openshift-4.19', got '%s'", siteConfig.Spec.ClusterImageSetNameRef)
	}

	// Verify non-supported fields are present (these should trigger warnings)
	if siteConfig.Spec.SshPrivateKeySecretRef.Name == "" {
		t.Error("Expected sshPrivateKeySecretRef to be set in comprehensive sample")
	}
	if len(siteConfig.Spec.CrTemplates) == 0 {
		t.Error("Expected crTemplates to be set at SiteConfig spec level in comprehensive sample")
	}
	if siteConfig.Spec.BiosConfigRef.FilePath == "" {
		t.Error("Expected biosConfigRef to be set at SiteConfig spec level in comprehensive sample")
	}

	// Verify cluster configuration
	if len(siteConfig.Spec.Clusters) != 1 {
		t.Fatalf("Expected 1 cluster, got %d", len(siteConfig.Spec.Clusters))
	}

	cluster := siteConfig.Spec.Clusters[0]
	if cluster.ClusterName != "comprehensive-cluster" {
		t.Errorf("Expected cluster name 'comprehensive-cluster', got '%s'", cluster.ClusterName)
	}

	// Verify cluster has all the expected supported fields
	if cluster.NetworkType != "OVNKubernetes" {
		t.Errorf("Expected networkType 'OVNKubernetes', got '%s'", cluster.NetworkType)
	}
	if cluster.ApiVIP != "192.168.1.100" {
		t.Errorf("Expected apiVIP '192.168.1.100', got '%s'", cluster.ApiVIP)
	}
	if len(cluster.ApiVIPs) != 2 {
		t.Errorf("Expected 2 apiVIPs, got %d", len(cluster.ApiVIPs))
	}
	if cluster.PlatformType != "baremetal" {
		t.Errorf("Expected platformType 'baremetal', got '%s'", cluster.PlatformType)
	}
	if cluster.CPUArchitecture != "x86_64" {
		t.Errorf("Expected cpuArchitecture 'x86_64', got '%s'", cluster.CPUArchitecture)
	}

	// Verify cluster has non-supported fields (should trigger warnings)
	if cluster.ExtraManifestPath == "" {
		t.Error("Expected extraManifestPath to be set in comprehensive sample")
	}
	if cluster.ExtraManifests.SearchPaths == nil || len(*cluster.ExtraManifests.SearchPaths) == 0 {
		t.Error("Expected extraManifests.searchPaths to be set in comprehensive sample")
	}
	if cluster.SiteConfigMap.Name == "" {
		t.Error("Expected siteConfigMap to be set in comprehensive sample")
	}
	if !cluster.MergeDefaultMachineConfigs {
		t.Error("Expected mergeDefaultMachineConfigs to be true in comprehensive sample")
	}
	if len(cluster.CrTemplates) == 0 {
		t.Error("Expected crTemplates to be set at cluster level in comprehensive sample")
	}

	// Verify manifestsConfigMapRefs (should be converted to extraManifestsRefs)
	if len(cluster.ManifestsConfigMapRefs) != 2 {
		t.Errorf("Expected 2 manifestsConfigMapRefs, got %d", len(cluster.ManifestsConfigMapRefs))
	}

	// Verify nodes
	if len(cluster.Nodes) != 3 {
		t.Fatalf("Expected 3 nodes, got %d", len(cluster.Nodes))
	}

	// Convert to ClusterInstance
	clusterTemplateNamespace := "test-namespace"
	clusterTemplateName := "test-cluster-template"
	nodeTemplateNamespace := "test-namespace"
	nodeTemplateName := "test-node-template"

	clusterInstance := convertClusterToClusterInstance(
		siteConfig,
		cluster,
		createTemplateRefs(clusterTemplateNamespace, clusterTemplateName),
		createTemplateRefs(nodeTemplateNamespace, nodeTemplateName),
		[]LocalObjectReference{}, // No command line manifests for this test
		"",                       // No command line suppressed manifests
		&WarningsCollector{},
		0,
		"test-siteconfig.yaml",
	)

	// Verify ClusterInstance basic fields
	if clusterInstance.ApiVersion != "siteconfig.open-cluster-management.io/v1alpha1" {
		t.Errorf("Expected apiVersion 'siteconfig.open-cluster-management.io/v1alpha1', got '%s'", clusterInstance.ApiVersion)
	}
	if clusterInstance.Kind != "ClusterInstance" {
		t.Errorf("Expected kind 'ClusterInstance', got '%s'", clusterInstance.Kind)
	}
	if clusterInstance.Metadata.Name != "comprehensive-cluster" {
		t.Errorf("Expected metadata name 'comprehensive-cluster', got '%s'", clusterInstance.Metadata.Name)
	}
	if clusterInstance.Metadata.Namespace != "comprehensive-cluster" {
		t.Errorf("Expected metadata namespace 'comprehensive-cluster', got '%s'", clusterInstance.Metadata.Namespace)
	}

	// Verify spec fields were properly converted
	if clusterInstance.Spec.BaseDomain != "example.com" {
		t.Errorf("Expected baseDomain 'example.com', got '%s'", clusterInstance.Spec.BaseDomain)
	}
	if clusterInstance.Spec.PullSecretRef.Name != "pull-secret" {
		t.Errorf("Expected pullSecretRef name 'pull-secret', got '%s'", clusterInstance.Spec.PullSecretRef.Name)
	}
	if clusterInstance.Spec.ClusterImageSetNameRef != "cluster-specific-image-set" {
		t.Errorf("Expected cluster-specific image set override, got '%s'", clusterInstance.Spec.ClusterImageSetNameRef)
	}
	if clusterInstance.Spec.ClusterName != "comprehensive-cluster" {
		t.Errorf("Expected clusterName 'comprehensive-cluster', got '%s'", clusterInstance.Spec.ClusterName)
	}
	if clusterInstance.Spec.ClusterType != "HighlyAvailable" {
		t.Errorf("Expected clusterType 'HighlyAvailable' (derived from 3 nodes), got '%s'", clusterInstance.Spec.ClusterType)
	}
	if clusterInstance.Spec.NetworkType != "OVNKubernetes" {
		t.Errorf("Expected networkType 'OVNKubernetes', got '%s'", clusterInstance.Spec.NetworkType)
	}

	// Verify VIPs conversion
	if len(clusterInstance.Spec.ApiVIPs) != 2 {
		t.Errorf("Expected 2 apiVIPs, got %d", len(clusterInstance.Spec.ApiVIPs))
	} else {
		if clusterInstance.Spec.ApiVIPs[0] != "192.168.1.100" {
			t.Errorf("Expected first apiVIP '192.168.1.100', got '%s'", clusterInstance.Spec.ApiVIPs[0])
		}
		if clusterInstance.Spec.ApiVIPs[1] != "2001:db8::100" {
			t.Errorf("Expected second apiVIP '2001:db8::100', got '%s'", clusterInstance.Spec.ApiVIPs[1])
		}
	}

	if len(clusterInstance.Spec.IngressVIPs) != 2 {
		t.Errorf("Expected 2 ingressVIPs, got %d", len(clusterInstance.Spec.IngressVIPs))
	}

	// Verify network configuration
	if len(clusterInstance.Spec.ClusterNetwork) != 2 {
		t.Errorf("Expected 2 cluster networks (IPv4 + IPv6), got %d", len(clusterInstance.Spec.ClusterNetwork))
	}
	if len(clusterInstance.Spec.ServiceNetwork) != 2 {
		t.Errorf("Expected 2 service networks (IPv4 + IPv6), got %d", len(clusterInstance.Spec.ServiceNetwork))
	}
	if len(clusterInstance.Spec.MachineNetwork) != 2 {
		t.Errorf("Expected 2 machine networks (IPv4 + IPv6), got %d", len(clusterInstance.Spec.MachineNetwork))
	}

	// Verify platform fields
	if clusterInstance.Spec.PlatformType != "baremetal" {
		t.Errorf("Expected platformType 'baremetal', got '%s'", clusterInstance.Spec.PlatformType)
	}
	if clusterInstance.Spec.CPUArchitecture != "x86_64" {
		t.Errorf("Expected cpuArchitecture 'x86_64', got '%s'", clusterInstance.Spec.CPUArchitecture)
	}
	if clusterInstance.Spec.CPUPartitioningMode != "AllNodes" {
		t.Errorf("Expected cpuPartitioningMode 'AllNodes', got '%s'", clusterInstance.Spec.CPUPartitioningMode)
	}

	// Verify labels transformation (clusterLabels -> extraLabels.ManagedCluster)
	if clusterInstance.Spec.ExtraLabels == nil {
		t.Error("Expected extraLabels to be set")
	} else {
		managedClusterLabels, exists := clusterInstance.Spec.ExtraLabels["ManagedCluster"]
		if !exists {
			t.Error("Expected extraLabels.ManagedCluster to exist")
		} else {
			expectedLabels := map[string]string{
				"environment":  "production",
				"region":       "us-west-2",
				"cluster-type": "edge",
				"workload":     "telco",
			}
			for key, expectedValue := range expectedLabels {
				if actualValue, exists := managedClusterLabels[key]; !exists {
					t.Errorf("Expected label '%s' to exist in ManagedCluster labels", key)
				} else if actualValue != expectedValue {
					t.Errorf("Expected label '%s' value '%s', got '%s'", key, expectedValue, actualValue)
				}
			}
		}
	}

	// Verify annotations transformation (crAnnotations.add -> extraAnnotations)
	if clusterInstance.Spec.ExtraAnnotations == nil {
		t.Error("Expected extraAnnotations to be set")
	} else {
		// Check for ManagedCluster annotations
		if managedClusterAnnotations, exists := clusterInstance.Spec.ExtraAnnotations["ManagedCluster"]; !exists {
			t.Error("Expected extraAnnotations.ManagedCluster to exist")
		} else {
			if managedClusterAnnotations["cluster.annotation/owner"] != "team-edge" {
				t.Error("Expected ManagedCluster annotation 'cluster.annotation/owner' to be 'team-edge'")
			}
		}
	}

	// Verify manifestsConfigMapRefs conversion to extraManifestsRefs
	if len(clusterInstance.Spec.ExtraManifestsRefs) != 2 {
		t.Errorf("Expected 2 extraManifestsRefs (from manifestsConfigMapRefs), got %d", len(clusterInstance.Spec.ExtraManifestsRefs))
	} else {
		expectedRefs := []string{"cluster-manifests-cm", "telco-manifests-cm"}
		for i, expectedRef := range expectedRefs {
			if i >= len(clusterInstance.Spec.ExtraManifestsRefs) {
				t.Errorf("Expected extraManifestsRefs[%d] to be '%s', but array is too short", i, expectedRef)
			} else if clusterInstance.Spec.ExtraManifestsRefs[i].Name != expectedRef {
				t.Errorf("Expected extraManifestsRefs[%d] to be '%s', got '%s'", i, expectedRef, clusterInstance.Spec.ExtraManifestsRefs[i].Name)
			}
		}
	}

	// Verify crSuppression conversion to suppressedManifests
	if len(clusterInstance.Spec.SuppressedManifests) != 2 {
		t.Errorf("Expected 2 suppressedManifests (from crSuppression), got %d", len(clusterInstance.Spec.SuppressedManifests))
	} else {
		expectedSuppressed := []string{"ConfigMap", "Secret"}
		for i, expected := range expectedSuppressed {
			if clusterInstance.Spec.SuppressedManifests[i] != expected {
				t.Errorf("Expected suppressedManifests[%d] to be '%s', got '%s'", i, expected, clusterInstance.Spec.SuppressedManifests[i])
			}
		}
	}

	// Verify disk encryption (Tang should be converted, TPM2 should be ignored)
	if clusterInstance.Spec.DiskEncryption == nil {
		t.Error("Expected diskEncryption to be set")
	} else {
		if clusterInstance.Spec.DiskEncryption.Type != "nbde" {
			t.Errorf("Expected diskEncryption type 'nbde', got '%s'", clusterInstance.Spec.DiskEncryption.Type)
		}
		if len(clusterInstance.Spec.DiskEncryption.Tang) != 2 {
			t.Errorf("Expected 2 Tang servers, got %d", len(clusterInstance.Spec.DiskEncryption.Tang))
		}
	}

	// Verify proxy configuration
	if clusterInstance.Spec.Proxy == nil {
		t.Error("Expected proxy to be set")
	} else {
		if clusterInstance.Spec.Proxy.HTTPProxy != "http://proxy.example.com:8080" {
			t.Errorf("Expected httpProxy 'http://proxy.example.com:8080', got '%s'", clusterInstance.Spec.Proxy.HTTPProxy)
		}
	}

	// Verify nodes
	if len(clusterInstance.Spec.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(clusterInstance.Spec.Nodes))
	}

	// Verify first node (master-1) with comprehensive configuration
	if len(clusterInstance.Spec.Nodes) > 0 {
		masterNode := clusterInstance.Spec.Nodes[0]
		if masterNode.HostName != "master-1.example.com" {
			t.Errorf("Expected first node hostname 'master-1.example.com', got '%s'", masterNode.HostName)
		}
		if masterNode.Role != "master" {
			t.Errorf("Expected first node role 'master', got '%s'", masterNode.Role)
		}
		if masterNode.BmcAddress != "redfish://192.168.1.10/redfish/v1/Systems/1" {
			t.Errorf("Expected first node bmcAddress 'redfish://192.168.1.10/redfish/v1/Systems/1', got '%s'", masterNode.BmcAddress)
		}
		if masterNode.BootMode != "UEFI" {
			t.Errorf("Expected first node bootMode 'UEFI', got '%s'", masterNode.BootMode)
		}

		// Verify node labels
		if len(masterNode.NodeLabels) == 0 {
			t.Error("Expected first node to have nodeLabels")
		} else {
			if masterNode.NodeLabels["hardware-type"] != "dell-r740" {
				t.Error("Expected first node to have hardware-type label 'dell-r740'")
			}
		}

		// Verify node annotations (crAnnotations.add -> extraAnnotations)
		if masterNode.ExtraAnnotations == nil {
			t.Error("Expected first node to have extraAnnotations")
		} else {
			if bmhAnnotations, exists := masterNode.ExtraAnnotations["BareMetalHost"]; !exists {
				t.Error("Expected first node to have BareMetalHost extraAnnotations")
			} else {
				if bmhAnnotations["bmh.annotation/hardware-profile"] != "dell-r740" {
					t.Error("Expected first node BareMetalHost annotation 'bmh.annotation/hardware-profile' to be 'dell-r740'")
				}
			}
		}

		// Verify node suppressedManifests (crSuppression -> suppressedManifests)
		if len(masterNode.SuppressedManifests) != 2 {
			t.Errorf("Expected first node to have 2 suppressedManifests, got %d", len(masterNode.SuppressedManifests))
		}

		// Verify node network configuration
		if masterNode.NodeNetwork == nil {
			t.Error("Expected first node to have nodeNetwork configuration")
		} else {
			if len(masterNode.NodeNetwork.Interfaces) != 2 {
				t.Errorf("Expected first node to have 2 network interfaces, got %d", len(masterNode.NodeNetwork.Interfaces))
			}
		}
	}

	// Verify template references
	if len(clusterInstance.Spec.TemplateRefs) != 1 {
		t.Errorf("Expected 1 cluster template reference, got %d", len(clusterInstance.Spec.TemplateRefs))
	} else {
		if clusterInstance.Spec.TemplateRefs[0].Name != clusterTemplateName {
			t.Errorf("Expected cluster template name '%s', got '%s'", clusterTemplateName, clusterInstance.Spec.TemplateRefs[0].Name)
		}
		if clusterInstance.Spec.TemplateRefs[0].Namespace != clusterTemplateNamespace {
			t.Errorf("Expected cluster template namespace '%s', got '%s'", clusterTemplateNamespace, clusterInstance.Spec.TemplateRefs[0].Namespace)
		}
	}

	// Verify node template references
	for i, node := range clusterInstance.Spec.Nodes {
		if len(node.TemplateRefs) != 1 {
			t.Errorf("Expected node[%d] to have 1 template reference, got %d", i, len(node.TemplateRefs))
		} else {
			if node.TemplateRefs[0].Name != nodeTemplateName {
				t.Errorf("Expected node[%d] template name '%s', got '%s'", i, nodeTemplateName, node.TemplateRefs[0].Name)
			}
			if node.TemplateRefs[0].Namespace != nodeTemplateNamespace {
				t.Errorf("Expected node[%d] template namespace '%s', got '%s'", i, nodeTemplateNamespace, node.TemplateRefs[0].Namespace)
			}
		}
	}

	t.Log("Successfully tested comprehensive SiteConfig sample conversion")
	t.Log("✅ All supported fields were properly converted to ClusterInstance")
	t.Log("✅ Field transformations (labels, annotations, manifests) work correctly")
	t.Log("✅ Multi-node configuration with master/worker roles is handled properly")
	t.Log("✅ Network configuration including dual-stack IPv4/IPv6 is converted correctly")
	t.Log("✅ Security features (disk encryption, proxy) are properly handled")
}

func TestWarningsForNonConvertibleFields(t *testing.T) {
	// Create a SiteConfig with fields that should trigger warnings
	siteConfigWithWarnings := `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: test-warnings
  namespace: test-namespace
spec:
  baseDomain: example.com
  pullSecretRef:
    name: pull-secret
  clusterImageSetNameRef: img-set
  sshPublicKey: "ssh-rsa test-key"
  sshPrivateKeySecretRef:
    name: my-ssh-private-key-secret
  biosConfigRef:
    filePath: "/path/to/global-bios-config"
  crTemplates:
    SriovOperatorConfig: "global-sriov-config"
  clusters:
  - clusterName: test-cluster
    biosConfigRef:
      filePath: "/path/to/cluster-bios-config"
    crTemplates:
      SriovNetworkNodePolicy: "cluster-sriov-policy"
    mergeDefaultMachineConfigs: true
    extraManifestOnly: true
    extraManifestPath: "/path/to/manifests"
    extraManifests:
      searchPaths:
        - "/path/to/extra/manifests"
      filter:
        inclusionDefault: "include"
        exclude: ["excluded-manifest"]
    siteConfigMap:
      name: "my-site-config-map"
    diskEncryption:
      type: "nbde"
      tpm2:
        pcrList: "1,7"
    nodes:
    - hostName: node1.example.com
      bmcAddress: redfish://192.168.1.1/redfish/v1/Systems/1
      bmcCredentialsName:
        name: node1-secret
      bootMACAddress: AA:BB:CC:DD:EE:FF
      role: master
      diskPartition:
        - device: "/dev/sda"
          partitions:
            - mount_point: "/var/log"
              size: 10000
              start: 0
              file_system_format: "xfs"
      userData:
        customField: "customValue"
        anotherField: 42
      biosConfigRef:
        filePath: "/path/to/node1-bios-config"
      crTemplates:
        NodeFeatureDiscovery: "node1-nfd-config"
      cpuset: "0-3"
    - hostName: node2.example.com
      bmcAddress: redfish://192.168.1.2/redfish/v1/Systems/1
      bmcCredentialsName:
        name: node2-secret
      bootMACAddress: AA:BB:CC:DD:EE:22
      role: worker
      userData:
        workerData: "workerValue"
      biosConfigRef:
        filePath: "/path/to/node2-bios-config"
      crTemplates:
        SriovNetworkNodePolicy: "node2-sriov-policy"
`

	// Write to temp file
	tempFile := "test-warnings.yaml"
	err := os.WriteFile(tempFile, []byte(siteConfigWithWarnings), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp SiteConfig file: %v", err)
	}
	defer os.Remove(tempFile)

	// Read the SiteConfig
	siteConfig, err := readSiteConfig(tempFile)
	if err != nil {
		t.Fatalf("Failed to read SiteConfig: %v", err)
	}

	// Capture stdout to check for warning messages
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the conversion (which should print warnings)
	outputDir := "test-warnings-output"
	defer os.RemoveAll(outputDir)

	err = convertToClusterInstance(siteConfig, outputDir, "test-ns/test-template", "test-ns/test-node-template", "", "", false, false, "")
	if err != nil {
		t.Errorf("Conversion failed: %v", err)
	}

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout

	capturedBytes, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}
	capturedOutput := string(capturedBytes)

	// Check for expected warning messages
	expectedWarnings := []string{
		"WARNING: sshPrivateKeySecretRef field 'my-ssh-private-key-secret' is not supported in ClusterInstance and will be ignored",
		"WARNING: biosConfigRef field '/path/to/global-bios-config' at SiteConfig spec level is not supported in ClusterInstance and will be ignored. Please create a custom node template for HostFirmwareSettings and reference it through templateRefs instead.Any nodes which use that custom template will then get the bios settings indicated in that CR",
		"WARNING: crTemplates field at SiteConfig spec level is not supported in ClusterInstance and will be ignored. To provide custom CR templates please use ConfigMaps and reference them through templateRefs instead.",
		"WARNING: biosConfigRef field '/path/to/cluster-bios-config' at cluster level is not supported in ClusterInstance and will be ignored. Please create a custom node template for HostFirmwareSettings and reference it through templateRefs instead.Any nodes which use that custom template will then get the bios settings indicated in that CR",
		"WARNING: crTemplates field at cluster level is not supported in ClusterInstance and will be ignored. To provide custom CR templates please use ConfigMaps and reference them through templateRefs instead.",
		"WARNING: mergeDefaultMachineConfigs field is not supported in ClusterInstance and will be ignored. Use a ConfigMap which contains the already merged MachineConfigs and reference it through extraManifestsRefs instead.",
		"WARNING: extraManifestOnly field is not supported in ClusterInstance and will be ignored.",
		"WARNING: extraManifests field is not supported in ClusterInstance and will be ignored. Create one or more configmaps with the exact desired set of CRs for the cluster and include them in the extraManifestsRefs.",
		"WARNING: extraManifestPath field '/path/to/manifests' is not supported in ClusterInstance and will be ignored. Create one or more configmaps with the exact desired set of CRs for the cluster and include them in the extraManifestsRefs.",
		"WARNING: siteConfigMap field 'my-site-config-map' is not supported in ClusterInstance and will be ignored. Create the site specific ConfigMap and place in git as a separate resource.",
		"WARNING: tpm2 disk encryption configuration is not supported in ClusterInstance and will be ignored. Conversion will be done only for the Tang server field.disk encryption MachineConfig with correct parameters must be added directly to the extramanifests configmap",
		"WARNING: diskPartition field on node 'node1.example.com' is not supported in ClusterInstance and will be ignored. Consider using IgnitionConfigOverride at the node level to configure disk partitions instead.",
		"WARNING: userData field on node 'node1.example.com' is not supported in ClusterInstance and will be ignored.Add userData through custom templates which add the necessary field to BareMetalHost",
		"WARNING: biosConfigRef field '/path/to/node1-bios-config' on node 'node1.example.com' is not supported in ClusterInstance and will be ignored. Please create a custom node template that includes HostFirmwareSettings and reference it through templateRefs instead.Any nodes which use that custom template will then get the bios settings indicated in that CR",
		"WARNING: crTemplates field on node 'node1.example.com' is not supported in ClusterInstance and will be ignored. To provide custom CR templates please use ConfigMaps and reference them through templateRefs instead.",
		"WARNING: cpuset field '0-3' on node 'node1.example.com' is not supported in ClusterInstance and will be ignored. Please see Workload Partitioning Feature for setting specific reserved/isolated CPUSets.",
		"WARNING: userData field on node 'node2.example.com' is not supported in ClusterInstance and will be ignored.Add userData through custom templates which add the necessary field to BareMetalHost",
		"WARNING: biosConfigRef field '/path/to/node2-bios-config' on node 'node2.example.com' is not supported in ClusterInstance and will be ignored. Please create a custom node template that includes HostFirmwareSettings and reference it through templateRefs instead.Any nodes which use that custom template will then get the bios settings indicated in that CR",
		"WARNING: crTemplates field on node 'node2.example.com' is not supported in ClusterInstance and will be ignored. To provide custom CR templates please use ConfigMaps and reference them through templateRefs instead.",
	}

	for _, expectedWarning := range expectedWarnings {
		if !strings.Contains(capturedOutput, expectedWarning) {
			t.Errorf("Expected warning not found in output: %s\nFull output: %s", expectedWarning, capturedOutput)
		}
	}

	// Verify that conversion still succeeds despite warnings
	if !strings.Contains(capturedOutput, "Successfully converted 1 cluster(s) to ClusterInstance files") {
		t.Error("Expected successful conversion message not found")
	}

	t.Log("All expected warnings were printed correctly")
}

func TestParseTemplateReferences(t *testing.T) {
	// Test cases for parseTemplateReferences function
	testCases := []struct {
		name          string
		input         string
		expectedRefs  []TemplateRef
		expectedError bool
		errorMessage  string
		description   string
	}{
		{
			name:          "Single template reference",
			input:         "test-namespace/test-template",
			expectedRefs:  []TemplateRef{{Name: "test-template", Namespace: "test-namespace"}},
			expectedError: false,
			description:   "Should parse single template reference correctly",
		},
		{
			name:  "Multiple template references",
			input: "ns1/template1,ns2/template2,ns3/template3",
			expectedRefs: []TemplateRef{
				{Name: "template1", Namespace: "ns1"},
				{Name: "template2", Namespace: "ns2"},
				{Name: "template3", Namespace: "ns3"},
			},
			expectedError: false,
			description:   "Should parse multiple template references correctly",
		},
		{
			name:          "Empty string",
			input:         "",
			expectedRefs:  []TemplateRef{},
			expectedError: false,
			description:   "Should handle empty string correctly",
		},
		{
			name:  "Whitespace in references",
			input: " ns1/template1 , ns2/template2 , ns3/template3 ",
			expectedRefs: []TemplateRef{
				{Name: "template1", Namespace: "ns1"},
				{Name: "template2", Namespace: "ns2"},
				{Name: "template3", Namespace: "ns3"},
			},
			expectedError: false,
			description:   "Should handle whitespace correctly",
		},
		{
			name:  "Empty entries in list",
			input: "ns1/template1,,ns2/template2,",
			expectedRefs: []TemplateRef{
				{Name: "template1", Namespace: "ns1"},
				{Name: "template2", Namespace: "ns2"},
			},
			expectedError: false,
			description:   "Should skip empty entries correctly",
		},
		{
			name:          "Invalid format - missing slash",
			input:         "invalid-template",
			expectedRefs:  nil,
			expectedError: true,
			errorMessage:  "invalid template reference format 'invalid-template', expected 'namespace/name'",
			description:   "Should reject template reference without slash",
		},
		{
			name:          "Invalid format - too many slashes",
			input:         "ns1/template1/extra",
			expectedRefs:  nil,
			expectedError: true,
			errorMessage:  "invalid template reference format 'ns1/template1/extra', expected 'namespace/name'",
			description:   "Should reject template reference with too many slashes",
		},
		{
			name:          "Invalid format - empty namespace",
			input:         "/template1",
			expectedRefs:  nil,
			expectedError: true,
			errorMessage:  "invalid template reference format '/template1', namespace and name cannot be empty",
			description:   "Should reject template reference with empty namespace",
		},
		{
			name:          "Invalid format - empty name",
			input:         "ns1/",
			expectedRefs:  nil,
			expectedError: true,
			errorMessage:  "invalid template reference format 'ns1/', namespace and name cannot be empty",
			description:   "Should reject template reference with empty name",
		},
		{
			name:          "Mixed valid and invalid",
			input:         "ns1/template1,invalid-template",
			expectedRefs:  nil,
			expectedError: true,
			errorMessage:  "invalid template reference format 'invalid-template', expected 'namespace/name'",
			description:   "Should reject entire list if any entry is invalid",
		},
		{
			name:  "Whitespace in namespace and name",
			input: " ns1 / template1 , ns2 / template2 ",
			expectedRefs: []TemplateRef{
				{Name: "template1", Namespace: "ns1"},
				{Name: "template2", Namespace: "ns2"},
			},
			expectedError: false,
			description:   "Should handle whitespace around namespace and name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			refs, err := parseTemplateReferences(tc.input)

			if tc.expectedError {
				if err == nil {
					t.Errorf("Expected error for input '%s', but got none", tc.input)
					return
				}
				if tc.errorMessage != "" && err.Error() != tc.errorMessage {
					t.Errorf("Expected error message '%s', got '%s'", tc.errorMessage, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input '%s': %v", tc.input, err)
				return
			}

			if len(refs) != len(tc.expectedRefs) {
				t.Errorf("Expected %d template references, got %d", len(tc.expectedRefs), len(refs))
				return
			}

			for i, expected := range tc.expectedRefs {
				if i >= len(refs) {
					t.Errorf("Expected template reference at index %d, but array is too short", i)
					continue
				}
				if refs[i].Name != expected.Name {
					t.Errorf("Expected template reference[%d] name '%s', got '%s'", i, expected.Name, refs[i].Name)
				}
				if refs[i].Namespace != expected.Namespace {
					t.Errorf("Expected template reference[%d] namespace '%s', got '%s'", i, expected.Namespace, refs[i].Namespace)
				}
			}

			t.Logf("Test %s passed: %s", tc.name, tc.description)
		})
	}
}

func TestMultipleTemplateReferencesConversion(t *testing.T) {
	// Test conversion with multiple template references
	siteConfig := &SiteConfig{
		ApiVersion: "ran.openshift.io/v1",
		Kind:       "SiteConfig",
		Metadata: Metadata{
			Name:      "test-multiple-templates",
			Namespace: "test-namespace",
		},
		Spec: Spec{
			BaseDomain: "example.com",
			PullSecretRef: PullSecretRef{
				Name: "pull-secret",
			},
			ClusterImageSetNameRef: "img-set",
			SshPublicKey:           "ssh-rsa test-key",
			Clusters: []Cluster{
				{
					ClusterName: "test-cluster",
					Nodes: []Node{
						{
							HostName:   "node1.example.com",
							BmcAddress: "redfish://192.168.1.1/redfish/v1/Systems/1",
							BmcCredentialsName: BmcCredentialsName{
								Name: "node1-secret",
							},
							BootMACAddress: "AA:BB:CC:DD:EE:FF",
							Role:           "master",
						},
					},
				},
			},
		},
	}

	cluster := siteConfig.Spec.Clusters[0]

	// Test with multiple cluster and node templates
	clusterTemplateRefs := []TemplateRef{
		{Name: "cluster-template-1", Namespace: "template-ns-1"},
		{Name: "cluster-template-2", Namespace: "template-ns-2"},
		{Name: "cluster-template-3", Namespace: "template-ns-3"},
	}

	nodeTemplateRefs := []TemplateRef{
		{Name: "node-template-1", Namespace: "node-ns-1"},
		{Name: "node-template-2", Namespace: "node-ns-2"},
	}

	clusterInstance := convertClusterToClusterInstance(
		siteConfig,
		cluster,
		clusterTemplateRefs,
		nodeTemplateRefs,
		[]LocalObjectReference{},
		"",
		&WarningsCollector{},
		0,
		"test-siteconfig.yaml",
	)

	// Verify cluster template references
	if len(clusterInstance.Spec.TemplateRefs) != len(clusterTemplateRefs) {
		t.Errorf("Expected %d cluster template references, got %d", len(clusterTemplateRefs), len(clusterInstance.Spec.TemplateRefs))
	} else {
		for i, expected := range clusterTemplateRefs {
			if clusterInstance.Spec.TemplateRefs[i].Name != expected.Name {
				t.Errorf("Expected cluster template[%d] name '%s', got '%s'", i, expected.Name, clusterInstance.Spec.TemplateRefs[i].Name)
			}
			if clusterInstance.Spec.TemplateRefs[i].Namespace != expected.Namespace {
				t.Errorf("Expected cluster template[%d] namespace '%s', got '%s'", i, expected.Namespace, clusterInstance.Spec.TemplateRefs[i].Namespace)
			}
		}
	}

	// Verify node template references
	if len(clusterInstance.Spec.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(clusterInstance.Spec.Nodes))
	}

	node := clusterInstance.Spec.Nodes[0]
	if len(node.TemplateRefs) != len(nodeTemplateRefs) {
		t.Errorf("Expected %d node template references, got %d", len(nodeTemplateRefs), len(node.TemplateRefs))
	} else {
		for i, expected := range nodeTemplateRefs {
			if node.TemplateRefs[i].Name != expected.Name {
				t.Errorf("Expected node template[%d] name '%s', got '%s'", i, expected.Name, node.TemplateRefs[i].Name)
			}
			if node.TemplateRefs[i].Namespace != expected.Namespace {
				t.Errorf("Expected node template[%d] namespace '%s', got '%s'", i, expected.Namespace, node.TemplateRefs[i].Namespace)
			}
		}
	}

	t.Log("Successfully tested multiple template references conversion")
	t.Logf("✅ Cluster templates: %d references correctly assigned", len(clusterTemplateRefs))
	t.Logf("✅ Node templates: %d references correctly assigned", len(nodeTemplateRefs))
}

func TestCommaSeparatedTemplateFlags(t *testing.T) {
	// Test the actual flag parsing with comma-separated values
	// Create a test SiteConfig YAML string
	siteConfigYAML := `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: test-comma-separated
  namespace: test-namespace
spec:
  baseDomain: example.com
  pullSecretRef:
    name: pull-secret
  clusterImageSetNameRef: img-set
  sshPublicKey: "ssh-rsa test-key"
  clusters:
  - clusterName: test-cluster
    nodes:
    - hostName: node1.example.com
      bmcAddress: redfish://192.168.1.1/redfish/v1/Systems/1
      bmcCredentialsName:
        name: node1-secret
      bootMACAddress: AA:BB:CC:DD:EE:FF
      role: master
`

	// Write to temp file
	tempFile := "test-comma-separated.yaml"
	err := os.WriteFile(tempFile, []byte(siteConfigYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp SiteConfig file: %v", err)
	}
	defer os.Remove(tempFile)

	// Read the SiteConfig
	siteConfig, err := readSiteConfig(tempFile)
	if err != nil {
		t.Fatalf("Failed to read SiteConfig: %v", err)
	}

	// Test comma-separated cluster templates
	clusterTemplateString := "cluster-ns-1/cluster-template-1,cluster-ns-2/cluster-template-2"
	clusterTemplateRefs, err := parseTemplateReferences(clusterTemplateString)
	if err != nil {
		t.Fatalf("Failed to parse cluster template references: %v", err)
	}

	// Test comma-separated node templates
	nodeTemplateString := "node-ns-1/node-template-1,node-ns-2/node-template-2,node-ns-3/node-template-3"
	nodeTemplateRefs, err := parseTemplateReferences(nodeTemplateString)
	if err != nil {
		t.Fatalf("Failed to parse node template references: %v", err)
	}

	// Test conversion
	err = convertToClusterInstance(siteConfig, "test-comma-separated-output", clusterTemplateString, nodeTemplateString, "", "", false, false, "")
	if err != nil {
		t.Fatalf("Failed to convert SiteConfig: %v", err)
	}
	defer os.RemoveAll("test-comma-separated-output")

	// Verify that the output file was created
	outputFile := "test-comma-separated-output/test-cluster.yaml"
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("Expected output file '%s' was not created", outputFile)
		return
	}

	// Read and verify the output ClusterInstance
	outputData, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var clusterInstance ClusterInstance
	err = yaml.Unmarshal(outputData, &clusterInstance)
	if err != nil {
		t.Fatalf("Failed to unmarshal ClusterInstance YAML: %v", err)
	}

	// Verify cluster template references
	expectedClusterTemplates := []TemplateRef{
		{Name: "cluster-template-1", Namespace: "cluster-ns-1"},
		{Name: "cluster-template-2", Namespace: "cluster-ns-2"},
	}

	if len(clusterInstance.Spec.TemplateRefs) != len(expectedClusterTemplates) {
		t.Errorf("Expected %d cluster template references, got %d", len(expectedClusterTemplates), len(clusterInstance.Spec.TemplateRefs))
	} else {
		for i, expected := range expectedClusterTemplates {
			if clusterInstance.Spec.TemplateRefs[i].Name != expected.Name {
				t.Errorf("Expected cluster template[%d] name '%s', got '%s'", i, expected.Name, clusterInstance.Spec.TemplateRefs[i].Name)
			}
			if clusterInstance.Spec.TemplateRefs[i].Namespace != expected.Namespace {
				t.Errorf("Expected cluster template[%d] namespace '%s', got '%s'", i, expected.Namespace, clusterInstance.Spec.TemplateRefs[i].Namespace)
			}
		}
	}

	// Verify node template references
	expectedNodeTemplates := []TemplateRef{
		{Name: "node-template-1", Namespace: "node-ns-1"},
		{Name: "node-template-2", Namespace: "node-ns-2"},
		{Name: "node-template-3", Namespace: "node-ns-3"},
	}

	if len(clusterInstance.Spec.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(clusterInstance.Spec.Nodes))
	}

	node := clusterInstance.Spec.Nodes[0]
	if len(node.TemplateRefs) != len(expectedNodeTemplates) {
		t.Errorf("Expected %d node template references, got %d", len(expectedNodeTemplates), len(node.TemplateRefs))
	} else {
		for i, expected := range expectedNodeTemplates {
			if node.TemplateRefs[i].Name != expected.Name {
				t.Errorf("Expected node template[%d] name '%s', got '%s'", i, expected.Name, node.TemplateRefs[i].Name)
			}
			if node.TemplateRefs[i].Namespace != expected.Namespace {
				t.Errorf("Expected node template[%d] namespace '%s', got '%s'", i, expected.Namespace, node.TemplateRefs[i].Namespace)
			}
		}
	}

	t.Log("Successfully tested comma-separated template flags")
	t.Logf("✅ Cluster templates: %d references parsed and converted correctly", len(clusterTemplateRefs))
	t.Logf("✅ Node templates: %d references parsed and converted correctly", len(nodeTemplateRefs))
	t.Log("✅ End-to-end conversion with comma-separated flags works correctly")
}

func TestCopyCommentsToClusterInstance(t *testing.T) {
	// Create a temporary SiteConfig YAML file with comments
	tempFile := `# This is a comprehensive SiteConfig example
# Author: Test User
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  # This is the metadata section
  name: test-with-comments
  namespace: test-namespace
  # Additional labels
  labels:
    environment: test
spec:
  # Base domain for the cluster
  baseDomain: example.com
  # Pull secret reference for container images
  pullSecretRef:
    name: pull-secret
  # Cluster image set reference
  clusterImageSetNameRef: img-set
  # SSH public key for node access
  sshPublicKey: "ssh-rsa test-key"
  # Cluster definitions
  clusters:
  - # Cluster configuration
    clusterName: test-cluster
    # Network type for the cluster
    networkType: OVNKubernetes
    # Cluster network configuration
    clusterNetwork:
      - cidr: 10.128.0.0/14
        hostPrefix: 23
    # API virtual IP
    apiVIP: 192.168.1.100
    # Ingress virtual IP
    ingressVIP: 192.168.1.101
    # Node configuration
    nodes:
      - # Master node configuration
        hostName: master.example.com
        bmcAddress: redfish://192.168.1.1/redfish/v1/Systems/1
        bmcCredentialsName:
          name: master-bmc-secret
        bootMACAddress: AA:BB:CC:DD:EE:FF
        role: master`

	// Write to temporary file
	tempFilePath := "test-comments-input.yaml"
	err := os.WriteFile(tempFilePath, []byte(tempFile), 0644)
	if err != nil {
		t.Fatalf("Failed to write temporary file: %v", err)
	}
	defer os.Remove(tempFilePath)

	// Parse the SiteConfig
	siteConfig, err := readSiteConfig(tempFilePath)
	if err != nil {
		t.Fatalf("Failed to parse SiteConfig: %v", err)
	}

	// Test with comments enabled
	outputDirWithComments := "test-comments-enabled"
	defer os.RemoveAll(outputDirWithComments)

	err = convertToClusterInstance(siteConfig, outputDirWithComments, "cluster-ns/cluster-template", "node-ns/node-template", "", "", false, true, tempFilePath)
	if err != nil {
		t.Fatalf("Failed to convert with comments enabled: %v", err)
	}

	// Read the generated ClusterInstance file
	clusterInstanceFile := filepath.Join(outputDirWithComments, "test-cluster.yaml")
	content, err := os.ReadFile(clusterInstanceFile)
	if err != nil {
		t.Fatalf("Failed to read ClusterInstance file: %v", err)
	}

	contentStr := string(content)

	// Check that field-specific comments are present
	expectedFieldComments := []string{
		"# Base domain for the cluster",
		"# Pull secret reference for container images",
		"# Cluster image set reference",
		"# SSH public key for node access",
		"# Cluster configuration",
		"# Network type for the cluster",
		"# Cluster network configuration",
		"# Node configuration",
	}

	for _, comment := range expectedFieldComments {
		if !strings.Contains(contentStr, comment) {
			t.Errorf("Expected field comment '%s' not found in ClusterInstance file", comment)
		}
	}

	// Check that header comments are present
	expectedHeaderComments := []string{
		"# Comments from original SiteConfig:",
		"# This is a comprehensive SiteConfig example",
		"# Author: Test User",
		"# This is the metadata section",
		"# Additional labels",
		"# Cluster definitions",
	}

	for _, comment := range expectedHeaderComments {
		if !strings.Contains(contentStr, comment) {
			t.Errorf("Expected header comment '%s' not found in ClusterInstance file", comment)
		}
	}

	// Test with comments disabled
	outputDirWithoutComments := "test-comments-disabled"
	defer os.RemoveAll(outputDirWithoutComments)

	err = convertToClusterInstance(siteConfig, outputDirWithoutComments, "cluster-ns/cluster-template", "node-ns/node-template", "", "", false, false, tempFilePath)
	if err != nil {
		t.Fatalf("Failed to convert with comments disabled: %v", err)
	}

	// Read the generated ClusterInstance file
	clusterInstanceFileNoComments := filepath.Join(outputDirWithoutComments, "test-cluster.yaml")
	contentNoComments, err := os.ReadFile(clusterInstanceFileNoComments)
	if err != nil {
		t.Fatalf("Failed to read ClusterInstance file without comments: %v", err)
	}

	contentNoCommentsStr := string(contentNoComments)

	// Check that the original SiteConfig comments are NOT present (only the standard extraManifestsRefs comment should be there)
	originalFieldComments := []string{
		"# Base domain for the cluster",
		"# Pull secret reference for container images",
		"# Cluster image set reference",
		"# SSH public key for node access",
		"# Cluster configuration",
		"# Network type for the cluster",
		"# Cluster network configuration",
		"# Node configuration",
	}

	for _, comment := range originalFieldComments {
		if strings.Contains(contentNoCommentsStr, comment) {
			t.Errorf("Original field comment '%s' should not be present when comments are disabled", comment)
		}
	}

	// Check that header comments are also NOT present when comments are disabled
	originalHeaderComments := []string{
		"# Comments from original SiteConfig:",
		"# This is a comprehensive SiteConfig example",
		"# Author: Test User",
		"# This is the metadata section",
		"# Additional labels",
		"# Cluster definitions",
	}

	for _, comment := range originalHeaderComments {
		if strings.Contains(contentNoCommentsStr, comment) {
			t.Errorf("Original header comment '%s' should not be present when comments are disabled", comment)
		}
	}

	// Note: No standard comments are expected when comments are disabled
}

func TestNodeLevelCommentsCopy(t *testing.T) {
	// Create a temporary SiteConfig YAML file with node-level comments
	tempFile := `apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: test-node-comments
  namespace: test-namespace
spec:
  baseDomain: example.com
  pullSecretRef:
    name: pull-secret
  clusterImageSetNameRef: img-set
  sshPublicKey: "ssh-rsa test-key"
  clusters:
  - clusterName: test-cluster
    networkType: OVNKubernetes
    nodes:
      - # Master node configuration
        hostName: master.example.com
        # BMC address with commented biosConfigRef
        #biosConfigRef:
        #  filePath: "path/to/bios-config.yaml"
        bmcAddress: redfish://192.168.1.1/redfish/v1/Systems/1
        # BMC credentials
        bmcCredentialsName:
          name: master-bmc-secret
        # Boot MAC address
        bootMACAddress: AA:BB:CC:DD:EE:FF
        # Node role
        role: master`

	// Write to temporary file
	tempFilePath := "test-node-comments-input.yaml"
	err := os.WriteFile(tempFilePath, []byte(tempFile), 0644)
	if err != nil {
		t.Fatalf("Failed to write temporary file: %v", err)
	}
	defer os.Remove(tempFilePath)

	// Parse the SiteConfig
	siteConfig, err := readSiteConfig(tempFilePath)
	if err != nil {
		t.Fatalf("Failed to parse SiteConfig: %v", err)
	}

	// Test with comments enabled
	outputDir := "test-node-comments-output"
	defer os.RemoveAll(outputDir)

	err = convertToClusterInstance(siteConfig, outputDir, "cluster-ns/cluster-template", "node-ns/node-template", "", "", false, true, tempFilePath)
	if err != nil {
		t.Fatalf("Failed to convert with comments enabled: %v", err)
	}

	// Read the generated ClusterInstance file
	clusterInstanceFile := filepath.Join(outputDir, "test-cluster.yaml")
	content, err := os.ReadFile(clusterInstanceFile)
	if err != nil {
		t.Fatalf("Failed to read ClusterInstance file: %v", err)
	}

	contentStr := string(content)

	// Check that node-level comments are present
	expectedNodeComments := []string{
		"# Master node configuration",
		"# BMC address with commented biosConfigRef",
		"# biosConfigRef:",
		"# filePath: \"path/to/bios-config.yaml\"",
		"# BMC credentials",
		"# Boot MAC address",
		"# Node role",
	}

	for _, comment := range expectedNodeComments {
		if !strings.Contains(contentStr, comment) {
			t.Errorf("Expected node comment '%s' not found in ClusterInstance file", comment)
		}
	}

	// Check that the biosConfigRef comment is placed near the bmcAddress field
	lines := strings.Split(contentStr, "\n")
	var biosConfigRefLineIndex, bmcAddressLineIndex int
	for i, line := range lines {
		if strings.Contains(line, "# biosConfigRef:") {
			biosConfigRefLineIndex = i
		}
		if strings.Contains(line, "bmcAddress:") {
			bmcAddressLineIndex = i
		}
	}

	// biosConfigRef comment should be within 3 lines before bmcAddress
	if biosConfigRefLineIndex == 0 || bmcAddressLineIndex == 0 {
		t.Error("Could not find biosConfigRef comment or bmcAddress field")
	} else if bmcAddressLineIndex-biosConfigRefLineIndex > 3 {
		t.Errorf("biosConfigRef comment (line %d) should be closer to bmcAddress field (line %d)", biosConfigRefLineIndex, bmcAddressLineIndex)
	}
}

func TestWriteWarningsToYAML(t *testing.T) {
	// Create a test SiteConfig YAML string with fields that generate warnings
	siteConfigYAML := `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: test-warnings
  namespace: test-namespace
spec:
  baseDomain: example.com
  pullSecretRef:
    name: pull-secret
  clusterImageSetNameRef: img-set
  sshPublicKey: "ssh-rsa test-key"
  biosConfigRef:
    filePath: /path/to/bios-config.yaml
  crTemplates:
    template1.yaml: |
      apiVersion: v1
      kind: ConfigMap
  clusters:
  - clusterName: test-cluster
    mergeDefaultMachineConfigs: true
    extraManifestOnly: true
    extraManifests:
      searchPaths:
        - /path/to/manifests
    extraManifestPath: /path/to/extra-manifest.yaml
    siteConfigMap:
      name: test-site-config
    diskEncryption:
      tpm2:
        pcrList: "1,7"
    nodes:
    - hostName: node1.example.com
      bmcAddress: redfish://192.168.1.1/redfish/v1/Systems/1
      bmcCredentialsName:
        name: node1-secret
      bootMACAddress: AA:BB:CC:DD:EE:FF
      role: master
      biosConfigRef:
        filePath: /path/to/node-bios-config.yaml
      crTemplates:
        node-template.yaml: |
          apiVersion: v1
          kind: ConfigMap
      userData:
        name: test-user-data
      diskPartition:
        - path: /dev/sda
          size: 100
      cpuset: "0-3"
`

	// Write to temp file
	tempFile := "test-warnings.yaml"
	err := os.WriteFile(tempFile, []byte(siteConfigYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp SiteConfig file: %v", err)
	}
	defer os.Remove(tempFile)

	// Read the SiteConfig
	siteConfig, err := readSiteConfig(tempFile)
	if err != nil {
		t.Fatalf("Failed to read SiteConfig: %v", err)
	}

	// Test conversion with writeWarnings=true
	outputDirWithWarnings := "test-warnings-output-with-warnings"
	err = convertToClusterInstance(siteConfig, outputDirWithWarnings, "cluster-ns/cluster-template", "node-ns/node-template", "", "", true, false, tempFile)
	if err != nil {
		t.Fatalf("Failed to convert SiteConfig with writeWarnings=true: %v", err)
	}
	defer os.RemoveAll(outputDirWithWarnings)

	// Test conversion with writeWarnings=false
	outputDirWithoutWarnings := "test-warnings-output-without-warnings"
	err = convertToClusterInstance(siteConfig, outputDirWithoutWarnings, "cluster-ns/cluster-template", "node-ns/node-template", "", "", false, false, tempFile)
	if err != nil {
		t.Fatalf("Failed to convert SiteConfig with writeWarnings=false: %v", err)
	}
	defer os.RemoveAll(outputDirWithoutWarnings)

	// Check that warnings were written to the YAML file when writeWarnings=true
	outputFileWithWarnings := outputDirWithWarnings + "/test-cluster.yaml"
	if _, err := os.Stat(outputFileWithWarnings); os.IsNotExist(err) {
		t.Errorf("Expected output file '%s' was not created", outputFileWithWarnings)
		return
	}

	contentWithWarnings, err := os.ReadFile(outputFileWithWarnings)
	if err != nil {
		t.Fatalf("Failed to read output file with warnings: %v", err)
	}

	contentWithWarningsStr := string(contentWithWarnings)
	if !strings.Contains(contentWithWarningsStr, "# Conversion Warnings:") {
		t.Errorf("Expected warnings comments to be present in YAML file with writeWarnings=true")
	}

	// Check that specific warnings are present
	expectedWarnings := []string{
		"biosConfigRef field",
		"crTemplates field",
		"mergeDefaultMachineConfigs field",
		"extraManifestOnly field",
		"extraManifests field",
		"extraManifestPath field",
		"siteConfigMap field",
		"tpm2 disk encryption",
		"diskPartition field",
		"userData field",
		"cpuset field",
	}

	for _, warning := range expectedWarnings {
		if !strings.Contains(contentWithWarningsStr, warning) {
			t.Errorf("Expected warning '%s' to be present in YAML file", warning)
		}
	}

	// Check that warnings were NOT written to the YAML file when writeWarnings=false
	outputFileWithoutWarnings := outputDirWithoutWarnings + "/test-cluster.yaml"
	if _, err := os.Stat(outputFileWithoutWarnings); os.IsNotExist(err) {
		t.Errorf("Expected output file '%s' was not created", outputFileWithoutWarnings)
		return
	}

	contentWithoutWarnings, err := os.ReadFile(outputFileWithoutWarnings)
	if err != nil {
		t.Fatalf("Failed to read output file without warnings: %v", err)
	}

	contentWithoutWarningsStr := string(contentWithoutWarnings)
	if strings.Contains(contentWithoutWarningsStr, "# Conversion Warnings:") {
		t.Errorf("Expected warnings comments to NOT be present in YAML file with writeWarnings=false")
	}

	// Verify the YAML content is valid ClusterInstance
	var clusterInstance ClusterInstance
	err = yaml.Unmarshal(contentWithWarnings, &clusterInstance)
	if err != nil {
		t.Fatalf("Failed to unmarshal ClusterInstance YAML with warnings: %v", err)
	}

	if clusterInstance.Kind != "ClusterInstance" {
		t.Errorf("Expected kind 'ClusterInstance', got '%s'", clusterInstance.Kind)
	}

	t.Log("Successfully tested -w flag functionality")
	t.Log("✅ Warnings are written as comments to YAML files when writeWarnings=true")
	t.Log("✅ Warnings are NOT written to YAML files when writeWarnings=false")
	t.Log("✅ YAML files with warnings are still valid ClusterInstance resources")
}
