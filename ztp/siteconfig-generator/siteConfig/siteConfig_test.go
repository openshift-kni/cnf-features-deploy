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
  - clusterName: "expect-defaults"
  - clusterName: "not-default-values"
    clusterType: "standard"
    networkType: "OpenShiftSDN"
    numMasters: 3
  - clusterName: "set-to-defaults"
    clusterType: "sno"
    clusterProfile: "none"
    networkType: "OVNKubernetes"
    numMasters: 1
`
	siteConfig := SiteConfig{}
	_ = yaml.Unmarshal([]byte(input), &siteConfig)

	// Validate ClusterType
	assert.Equal(t, siteConfig.Spec.Clusters[0].ClusterType, "sno")
	assert.Equal(t, siteConfig.Spec.Clusters[1].ClusterType, "standard")
	assert.Equal(t, siteConfig.Spec.Clusters[2].ClusterType, "sno")

	// Validate NetworkType
	assert.Equal(t, siteConfig.Spec.Clusters[0].NetworkType, "OVNKubernetes")
	assert.Equal(t, siteConfig.Spec.Clusters[1].NetworkType, "OpenShiftSDN")
	assert.Equal(t, siteConfig.Spec.Clusters[2].NetworkType, "OVNKubernetes")

	// Validate NumMasters
	assert.Equal(t, siteConfig.Spec.Clusters[0].NumMasters, uint8(1))
	assert.Equal(t, siteConfig.Spec.Clusters[1].NumMasters, uint8(3))
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

func TestGetSiteConfigFieldValue(t *testing.T) {
	pullSecretValue := "pullSecretName"
	cluster0Node0BmcValue := "bmc-secret"
	cluster1Node1Name := "node1"
	siteConfigStr := `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "test-site"
  namespace: "test-site"
spec:
  pullSecretRef:
    name: ` + pullSecretValue + `
  clusters:
  - clusterName: "test-site0"
    extraManifestPath: testSiteConfig/testUserExtraManifest
    nodes:
      - hostName: "node0"
        bmcCredentialsName:
          name: ` + cluster0Node0BmcValue + `
  - clusterName: "test-site1"
    nodes:
      - hostName: "node0"
        bmcCredentialsName:
          name: "bmc-secret0"
      - hostName: ` + cluster1Node1Name

	siteConfig := SiteConfig{}
	_ = yaml.Unmarshal([]byte(siteConfigStr), &siteConfig)

	fieldV, _ := siteConfig.GetSiteConfigFieldValue("siteconfig.Spec.PullSecretRef.Name", 0, 0)
	assert.Equal(t, fieldV, pullSecretValue)

	fieldV, _ = siteConfig.GetSiteConfigFieldValue("siteconfig.Spec.Clusters.Nodes.BmcCredentialsName.Name", 0, 0)
	assert.Equal(t, fieldV, cluster0Node0BmcValue)

	fieldV, _ = siteConfig.GetSiteConfigFieldValue("siteconfig.Spec.Clusters.Nodes.HostName", 1, 1)
	assert.Equal(t, fieldV, cluster1Node1Name)

	// Test empty path
	fieldV, _ = siteConfig.GetSiteConfigFieldValue("siteconfig.Spec.Clusters.Nodes.BmcCredentialsName.Name", 1, 1)
	assert.Equal(t, fieldV, "")

	// Test wrong path
	fieldV, _ = siteConfig.GetSiteConfigFieldValue("siteconfig.Spec.Wrong.Path", 0, 0)
	assert.Equal(t, fieldV, nil)
}
