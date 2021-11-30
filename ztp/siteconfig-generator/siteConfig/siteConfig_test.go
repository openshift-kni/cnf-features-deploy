package siteConfig

import (
	"fmt"
	"strings"
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
    nodes:
    - hostName: "node1-default"
  - clusterName: "not-default-values"
    networkType: "OpenShiftSDN"
    nodes:
    - hostName: "node1-default"
  - clusterName: "set-to-defaults"
    networkType: "OVNKubernetes"
    nodes:
    - hostName: "node1-default"
`
	siteConfig := SiteConfig{}
	err := yaml.Unmarshal([]byte(input), &siteConfig)
	assert.NoError(t, err)

	// Validate NetworkType
	assert.Equal(t, siteConfig.Spec.Clusters[0].NetworkType, "OVNKubernetes")
	assert.Equal(t, siteConfig.Spec.Clusters[1].NetworkType, "OpenShiftSDN")
	assert.Equal(t, siteConfig.Spec.Clusters[2].NetworkType, "OVNKubernetes")
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
    - hostName: "master1-default"
    - hostName: "master2-explicit"
      bootMode: "UEFI"
      role: "master"
    - hostName: "master3-default"
`
	siteConfig := SiteConfig{}
	err := yaml.Unmarshal([]byte(input), &siteConfig)
	assert.NoError(t, err)

	// Validate BootMode
	assert.Equal(t, siteConfig.Spec.Clusters[0].Nodes[0].BootMode, "legacy")
	assert.Equal(t, siteConfig.Spec.Clusters[0].Nodes[1].BootMode, "UEFI")
	assert.Equal(t, siteConfig.Spec.Clusters[0].Nodes[2].BootMode, "UEFI")
	assert.Equal(t, siteConfig.Spec.Clusters[0].Nodes[3].BootMode, "UEFI")

	// Validate Role
	assert.Equal(t, siteConfig.Spec.Clusters[0].Nodes[0].Role, "worker")
	assert.Equal(t, siteConfig.Spec.Clusters[0].Nodes[1].Role, "master")
	assert.Equal(t, siteConfig.Spec.Clusters[0].Nodes[2].Role, "master")
	assert.Equal(t, siteConfig.Spec.Clusters[0].Nodes[3].Role, "master")
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
    nodes:
    - hostName: "master1"
    diskEncryption:
      type: nbde
  - clusterName: "defaults"
    nodes:
    - hostName: "master1"
    # Without further content under diskEncryption the type does not get populated
    diskEncryption:
      type:
  - clusterName: "explicit"
    nodes:
    - hostName: "master1"
    diskEncryption:
      type: none
`
	siteConfig := SiteConfig{}
	err := yaml.Unmarshal([]byte(input), &siteConfig)
	assert.NoError(t, err)

	// Validate Disk Encryption type
	assert.Equal(t, siteConfig.Spec.Clusters[0].DiskEncryption.Type, "nbde")
	assert.Equal(t, siteConfig.Spec.Clusters[1].DiskEncryption.Type, "none")
	assert.Equal(t, siteConfig.Spec.Clusters[2].DiskEncryption.Type, "none")
}

func TestRoleCounters(t *testing.T) {
	input := `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
spec:
  clusters:
  - clusterName: "ignore-user-supplied-values"
    numMasters: 5
    numWorkers: 22
    clusterType: "any value"
    nodes:
    - hostName: "node0"
  - clusterName: "sno"
    nodes:
    - hostName: "node0"
      # Default role is "master"
  - clusterName: "3-node"
    nodes:
    - hostName: "node0"
      role: "master"
    - hostName: "node1"
      role: "master"
    - hostName: "node2"
      role: "master"
  - clusterName: "standard"
    nodes:
    - hostName: "node0"
      role: "master"
    - hostName: "node1"
      role: "master"
    - hostName: "node2"
      role: "master"
    - hostName: "worker0"
      role: "worker"
    - hostName: "worker1"
      role: "worker"
`
	siteConfig := SiteConfig{}
	err := yaml.Unmarshal([]byte(input), &siteConfig)
	assert.NoError(t, err)

	// Validate NumMasters
	assert.Equal(t, siteConfig.Spec.Clusters[0].NumMasters, uint8(1))
	assert.Equal(t, siteConfig.Spec.Clusters[1].NumMasters, uint8(1))
	assert.Equal(t, siteConfig.Spec.Clusters[2].NumMasters, uint8(3))
	assert.Equal(t, siteConfig.Spec.Clusters[3].NumMasters, uint8(3))

	// Validate NumWorkers
	assert.Equal(t, siteConfig.Spec.Clusters[0].NumWorkers, uint8(0))
	assert.Equal(t, siteConfig.Spec.Clusters[1].NumWorkers, uint8(0))
	assert.Equal(t, siteConfig.Spec.Clusters[2].NumWorkers, uint8(0))
	assert.Equal(t, siteConfig.Spec.Clusters[3].NumWorkers, uint8(2))

	// Validate ClusterType
	assert.Equal(t, siteConfig.Spec.Clusters[0].ClusterType, "sno")
	assert.Equal(t, siteConfig.Spec.Clusters[1].ClusterType, "sno")
	assert.Equal(t, siteConfig.Spec.Clusters[2].ClusterType, "standard")
	assert.Equal(t, siteConfig.Spec.Clusters[3].ClusterType, "standard")

	// Failure cases: Wrong number of masters
	for _, i := range []int{0, 2, 4, 5, 10, 100} {
		badInput := `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
spec:
  clusters:
  - clusterName: "ignore-user-supplied-numbers"
    nodes:
`
		for j := 0; j < i; j++ {
			badInput = badInput + fmt.Sprintf("\n    - hostName: \"node%d\"", j)
		}
		err := yaml.Unmarshal([]byte(badInput), &siteConfig)
		assert.Error(t, err, "Expected an error with %d masters defined", i)
		assert.True(t, strings.Contains(err.Error(), fmt.Sprintf("(counted %d)", i)), "Expecting counted masters to match %d: %s", i, err.Error())
	}
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
      - hostName: ` + cluster1Node1Name + `
      - hostName: "node2"
`

	siteConfig := SiteConfig{}
	err := yaml.Unmarshal([]byte(siteConfigStr), &siteConfig)
	assert.NoError(t, err)

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
