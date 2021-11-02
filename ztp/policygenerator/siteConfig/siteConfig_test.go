package siteConfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// Test cases for default values on fields in the SiteConfig.Clusters[] entries
func TestClusterDefaults(t *testing.T) {
	input := `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
spec:
  clusters:
  - clusterName: "not-default-values"
    clusterType: "sno"
    clusterProfile: "du"
    networkType: "OpenShiftSDN"
    numMasters: 2
  - clusterName: "expect-defaults"
  - clusterName: "set-to-defaults"
    clusterType: "standard"
    clusterProfile: "none"
    networkType: "OVNKubernetes"
    numMasters: 1
`
	siteConfig := SiteConfig{}
	_ = yaml.Unmarshal([]byte(input), &siteConfig)

	// Validate ClusterType
	assert.Equal(t, siteConfig.Spec.Clusters[0].ClusterType, "sno")
	assert.Equal(t, siteConfig.Spec.Clusters[1].ClusterType, "standard")
	assert.Equal(t, siteConfig.Spec.Clusters[2].ClusterType, "standard")

	// Validate ClusterProfile
	assert.Equal(t, siteConfig.Spec.Clusters[0].ClusterProfile, "du")
	assert.Equal(t, siteConfig.Spec.Clusters[1].ClusterProfile, "none")
	assert.Equal(t, siteConfig.Spec.Clusters[2].ClusterProfile, "none")

	// Validate NetworkType
	assert.Equal(t, siteConfig.Spec.Clusters[0].NetworkType, "OpenShiftSDN")
	assert.Equal(t, siteConfig.Spec.Clusters[1].NetworkType, "OVNKubernetes")
	assert.Equal(t, siteConfig.Spec.Clusters[2].NetworkType, "OVNKubernetes")

	// Validate NumMasters
	assert.Equal(t, siteConfig.Spec.Clusters[0].NumMasters, uint8(2))
	assert.Equal(t, siteConfig.Spec.Clusters[1].NumMasters, uint8(1))
	assert.Equal(t, siteConfig.Spec.Clusters[2].NumMasters, uint8(1))
}

// Test cases for default values on fields in the SiteConfig.Clusters[].Nodes[]
// entries
func TestNodeDefaults(t *testing.T) {
	input := `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
spec:
  clusters:
  - clusterName: "just-for-testing-node-defaults"
    nodes:
    - hostName: "node0-not-default"
      bootMode: "legacy"
      role: "worker"
    - hostName: "node1-default"
    - hostName: "node2-explicit"
      bootMode: "UEFI"
      role: "master"
`
	siteConfig := SiteConfig{}
	_ = yaml.Unmarshal([]byte(input), &siteConfig)

	// Validate BootMode
	assert.Equal(t, siteConfig.Spec.Clusters[0].Nodes[0].BootMode, "legacy")
	assert.Equal(t, siteConfig.Spec.Clusters[0].Nodes[1].BootMode, "UEFI")
	assert.Equal(t, siteConfig.Spec.Clusters[0].Nodes[2].BootMode, "UEFI")

	// Validate Role
	assert.Equal(t, siteConfig.Spec.Clusters[0].Nodes[0].Role, "worker")
	assert.Equal(t, siteConfig.Spec.Clusters[0].Nodes[1].Role, "master")
	assert.Equal(t, siteConfig.Spec.Clusters[0].Nodes[2].Role, "master")
}

// Test cases for default values on fields in the
// SiteConfig.Clusters[].DiskEncryption entries
func TestNodeDiskEncryptionDefaults(t *testing.T) {
	input := `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
spec:
  clusters:
  - clusterName: "user"
    diskEncryption:
      type: nbde
  - clusterName: "defaults"
    # Without further content under diskEncryptionthe type does not get populated
    diskEncryption:
      type:
  - clusterName: "explicit"
    diskEncryption:
      type: none
`
	siteConfig := SiteConfig{}
	_ = yaml.Unmarshal([]byte(input), &siteConfig)

	// Validate ClusterType
	assert.Equal(t, siteConfig.Spec.Clusters[0].DiskEncryption.Type, "nbde")
	assert.Equal(t, siteConfig.Spec.Clusters[1].DiskEncryption.Type, "none")
	assert.Equal(t, siteConfig.Spec.Clusters[2].DiskEncryption.Type, "none")

}
