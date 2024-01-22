package siteConfig

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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
    siteConfigMap:
      data:
        key1: value1
    nodes:
    - hostName: "node1-default"
  - clusterName: "not-default-values"
    networkType: "OpenShiftSDN"
    siteConfigMap:
      name: not-default
      data:
        key1: value1
    nodes:
    - hostName: "node1-default"
  - clusterName: "set-to-defaults"
    networkType: "OVNKubernetes"
    siteConfigMap:
      namespace: not-default
      data:
        key1: value1
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
	// Validate siteConfigMap
	assert.Equal(t, siteConfig.Spec.Clusters[0].SiteConfigMap.Name, "")
	assert.Equal(t, siteConfig.Spec.Clusters[0].SiteConfigMap.Namespace, "ztp-site")
	assert.Equal(t, siteConfig.Spec.Clusters[1].SiteConfigMap.Name, "not-default")
	assert.Equal(t, siteConfig.Spec.Clusters[1].SiteConfigMap.Namespace, "ztp-site")
	assert.Equal(t, siteConfig.Spec.Clusters[2].SiteConfigMap.Name, "")
	assert.Equal(t, siteConfig.Spec.Clusters[2].SiteConfigMap.Namespace, "not-default")
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
      automatedCleaningMode: "metadata"
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

	// Validate AutomatedCleaningMode
	assert.Equal(t, siteConfig.Spec.Clusters[0].Nodes[0].AutomatedCleaningMode, "disabled")
	assert.Equal(t, siteConfig.Spec.Clusters[0].Nodes[1].AutomatedCleaningMode, "metadata")
	assert.Equal(t, siteConfig.Spec.Clusters[0].Nodes[2].AutomatedCleaningMode, "disabled")
	assert.Equal(t, siteConfig.Spec.Clusters[0].Nodes[3].AutomatedCleaningMode, "disabled")
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
    numControlPlanes: 5
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

	// Validate NumControlPlanes
	assert.Equal(t, siteConfig.Spec.Clusters[0].NumControlPlanes, uint8(1))
	assert.Equal(t, siteConfig.Spec.Clusters[1].NumControlPlanes, uint8(1))
	assert.Equal(t, siteConfig.Spec.Clusters[2].NumControlPlanes, uint8(3))
	assert.Equal(t, siteConfig.Spec.Clusters[3].NumControlPlanes, uint8(3))

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

	// Failure cases: Wrong number of masters(0)
	badInput := `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
spec:
  clusters:
  - clusterName: "ignore-user-supplied-numbers"
    nodes:
`
	err = yaml.Unmarshal([]byte(badInput), &siteConfig)
	assert.Error(t, err, "Expected an error with 0 control-planes defined")
	assert.True(t, strings.Contains(err.Error(), "must be 1 or more"), "Expecting counted control-planes to match %d:", 0, err.Error())
}

func TestGetSiteConfigFieldValue(t *testing.T) {
	pullSecretValue := "pullSecretName"
	cluster0Node0BmcValue := "bmc-secret"
	cluster0Node0AutomatedCleaningMode := "metadata"
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
        automatedCleaningMode: ` + cluster0Node0AutomatedCleaningMode + `
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

	fieldV, _ = siteConfig.GetSiteConfigFieldValue("siteconfig.Spec.Clusters.Nodes.AutomatedCleaningMode", 0, 0)
	assert.Equal(t, fieldV, cluster0Node0AutomatedCleaningMode)

	// Test empty path
	fieldV, _ = siteConfig.GetSiteConfigFieldValue("siteconfig.Spec.Clusters.Nodes.BmcCredentialsName.Name", 1, 1)
	assert.Equal(t, fieldV, "")

	// Test wrong path
	fieldV, _ = siteConfig.GetSiteConfigFieldValue("siteconfig.Spec.Wrong.Path", 0, 0)
	assert.Equal(t, fieldV, nil)
}

func TestCrTemplateSearch(t *testing.T) {
	input := `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
spec:
  crTemplates:
   a: site
   b: site
   c: site
   f: site
  clusters:
  - clusterName: "unset"
    crTemplates:
      b: cluster
      c: cluster
      d: cluster
      g: cluster
    nodes:
    - hostName: "unset"
      crTemplates:
        c: node
        d: node
        e: node
        f: node
`
	tests := []struct {
		key     string
		site    string
		cluster string
		node    string
	}{
		{key: "", site: "", cluster: "", node: ""},
		{key: "not found", site: "", cluster: "", node: ""},
		{key: "a", site: "site", cluster: "site", node: "site"},
		{key: "b", site: "site", cluster: "cluster", node: "cluster"},
		{key: "c", site: "site", cluster: "cluster", node: "node"},
		{key: "d", site: "", cluster: "cluster", node: "node"},
		{key: "e", site: "", cluster: "", node: "node"},
		{key: "f", site: "site", cluster: "site", node: "node"},
		{key: "g", site: "", cluster: "cluster", node: "cluster"},
	}

	siteConfig := SiteConfig{}
	err := yaml.Unmarshal([]byte(input), &siteConfig)
	assert.NoError(t, err)

	for _, test := range tests {
		site := siteConfig.Spec
		cluster := site.Clusters[0]
		node := cluster.Nodes[0]
		siteValue, siteOk := site.CrTemplateSearch(test.key)
		assertLookup(t, test.site, siteValue, siteOk, "site", test.key)
		clusterValue, clusterOk := cluster.CrTemplateSearch(test.key, &site)
		assertLookup(t, test.cluster, clusterValue, clusterOk, "cluster", test.key)
		nodeValue, nodeOk := node.CrTemplateSearch(test.key, &cluster, &site)
		assertLookup(t, test.node, nodeValue, nodeOk, "node", test.key)
	}
}

func assertLookup(t *testing.T, expected, actual string, ok bool, level, key string) {
	if expected == "" {
		assert.False(t, ok, "%s lookup of %s", level, key)
	} else {
		assert.True(t, ok, "%s lookup of %s", level, key)
		assert.Equal(t, expected, actual, "%s value of %s", level, key)
	}
}

func TestAllOverridesAreValid(t *testing.T) {
	input := `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
spec:
  crTemplates:
    site: site
  clusters:
  - clusterName: "unset"
    crTemplates:
      cluster: cluster
    nodes:
    - hostName: "unset"
      crTemplates:
        node: node
`

	tests := []struct {
		validKinds            map[string]bool
		validNodeKinds        map[string]bool
		expectedErrorLocation string
		expectedErrorKind     string
	}{{
		validKinds:            map[string]bool{"site": true, "cluster": true, "node": true},
		validNodeKinds:        map[string]bool{"node": true},
		expectedErrorLocation: "",
		expectedErrorKind:     "",
	}, {
		validKinds:            map[string]bool{"cluster": true, "node": true},
		validNodeKinds:        map[string]bool{"node": true},
		expectedErrorLocation: "SiteConfig.Spec",
		expectedErrorKind:     "site",
	}, {
		validKinds:            map[string]bool{"site": true, "node": true},
		validNodeKinds:        map[string]bool{"node": true},
		expectedErrorLocation: "SiteConfig.Spec.Clusters[0]",
		expectedErrorKind:     "cluster",
	}, {
		validKinds:            map[string]bool{"site": true, "cluster": true},
		validNodeKinds:        map[string]bool{},
		expectedErrorLocation: "SiteConfig.Spec.Clusters[0].Nodes[0]",
		expectedErrorKind:     "node",
	}, {
		validKinds:            map[string]bool{},
		validNodeKinds:        map[string]bool{},
		expectedErrorLocation: "SiteConfig.Spec",
		expectedErrorKind:     "site",
	}}

	siteConfig := SiteConfig{}
	err := yaml.Unmarshal([]byte(input), &siteConfig)
	assert.NoError(t, err)

	for _, test := range tests {
		err := siteConfig.areAllOverridesValid(&test.validKinds, &test.validNodeKinds)
		if test.expectedErrorLocation == "" {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), fmt.Sprintf("%s:", test.expectedErrorLocation))
			assert.Contains(t, err.Error(), fmt.Sprintf("%q is not a valid CR type", test.expectedErrorKind))
		}
	}
}

func TestBiosFileSearch(t *testing.T) {
	siteConfigStr := `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
spec:
  biosConfigRef:
    filePath: "site_file"
  clusters:
  - clusterName: "cluster1"
    biosConfigRef:
      filePath: cluster_file
    nodes:
      - hostName: "node1"
        biosConfigRef:
          filePath: "node_file"
      - hostName: "node2"
      - hostName: "node3"
  - clusterName: "cluster2"
    nodes:
      - hostName: "node1"
`
	siteConfig := SiteConfig{}
	err := yaml.Unmarshal([]byte(siteConfigStr), &siteConfig)
	assert.NoError(t, err)

	site := siteConfig.Spec

	cluster := site.Clusters[0]
	node := cluster.Nodes[0]
	nodeValue := node.BiosFileSearch(&cluster, &site)
	assert.Equal(t, nodeValue, "node_file")
	node = cluster.Nodes[1]
	nodeValue = node.BiosFileSearch(&cluster, &site)
	assert.Equal(t, nodeValue, "cluster_file")

	cluster = site.Clusters[1]
	node = cluster.Nodes[0]
	nodeValue = node.BiosFileSearch(&cluster, &site)
	assert.Equal(t, nodeValue, "site_file")
}

func TestPartitions_UnmarshalYAML(t *testing.T) {
	var inputFmt = `
mount_point: %s
size: %s
start: %s
file_system_format: %s
`

	type fields struct {
		MountPoint    string
		Size          int
		Start         int
		MountFileName string
		Label         string
	}
	tests := []struct {
		name           string
		fields         fields
		input          []byte
		wantErr        bool
		expectedResult *Partitions
		expectedError  string
	}{
		{
			name: "check if the mount file name is generated correctly",
			expectedResult: &Partitions{
				MountPoint:       "/var/imageregistry",
				Size:             100000,
				Start:            25000,
				MountFileName:    "var-imageregistry.mount",
				Label:            "var-imageregistry",
				FileSystemFormat: "xfs",
			},
			wantErr: false,
			input:   []byte(fmt.Sprintf(inputFmt, "/var/imageregistry", "100000", "25000", "")),
		},
		{
			name:          "expect error when start size is too small",
			input:         []byte(fmt.Sprintf(inputFmt, "/var/imageregistry", "100000", "0", "")),
			wantErr:       true,
			expectedError: "start value too small. must be over 25000",
		},
		{
			name:          "expect error when the partition size is too small",
			input:         []byte(fmt.Sprintf(inputFmt, "/var/imageregistry", "0", "25000", "")),
			wantErr:       true,
			expectedError: "choose an appropriate partition size. must be greater than 0",
		},
		{
			name:    "mount file name and labels are correctly generated",
			input:   []byte(fmt.Sprintf(inputFmt, "/my/path/another/dir", "100000", "25000", "")),
			wantErr: false,
			expectedResult: &Partitions{
				MountPoint:       "/my/path/another/dir",
				Size:             100000,
				Start:            25000,
				Label:            "my-path-another-dir",
				MountFileName:    "my-path-another-dir.mount",
				FileSystemFormat: "xfs",
			},
		},
		{
			name:    "use a different filesystem format by overriding the default",
			input:   []byte(fmt.Sprintf(inputFmt, "/my/path/another/dir", "100000", "25000", "mycustomformat")),
			wantErr: false,
			expectedResult: &Partitions{
				MountPoint:       "/my/path/another/dir",
				Size:             100000,
				Start:            25000,
				Label:            "my-path-another-dir",
				MountFileName:    "my-path-another-dir.mount",
				FileSystemFormat: "mycustomformat",
			},
		},
		{
			name:          "mount point is required and multiple error concatenated",
			input:         []byte(fmt.Sprintf(inputFmt, "var/imageregistry", "0", "25000", "")),
			wantErr:       true,
			expectedError: "choose an appropriate partition size. must be greater than 0 && path must be absolute mount_point. e.g /var/path",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prt := &Partitions{}
			err := yaml.Unmarshal(tt.input, prt)
			if !tt.wantErr {
				if !(cmp.Equal(prt, tt.expectedResult)) {
					t.Errorf("EXPECTED: %v, GOT: %v", tt.expectedResult, prt)
				}
			} else {
				if !(cmp.Equal(err, tt.expectedError)) {
					assert.EqualErrorf(t, err, tt.expectedError, "EXPECTED: %v, GOT: %v", tt.expectedError, err)
				}
			}
		})
	}
}

func Test_MustBeSnoToDiskPart(t *testing.T) {
	const siteConfigStr = `
spec:
  clusters:
  - clusterName: "cluster1"
    nodes:
      - hostName: "node1"
      - hostName: "node2"
        diskPartition:
           - device: "/dev/disk/by-path/pci-0000:01:00.0-scsi-0:2:0:0"
      - hostName: "node3"
`
	siteConfig := SiteConfig{}
	err := yaml.Unmarshal([]byte(siteConfigStr), &siteConfig)
	assert.Error(t, err)
}

func Test_Proxy(t *testing.T) {
	siteConfigStr := `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "test-site"
  namespace: "test-site"
spec:
  pullSecretRef:
    name: "pullSecretRef"
  clusters:
  - clusterName: "test-site0"
    extraManifestPath: testSiteConfig/testUserExtraManifest
    proxy:
      httpProxy: "test"
      httpsProxy: "test"
      noProxy: "test"
    nodes:
      - hostName: "node0"
  - clusterName: "test-site1"
    nodes:
      - hostName: "node0"
        bmcCredentialsName:
          name: "bmc-secret0"
      - hostName: "node2"
`
	siteConfig := SiteConfig{}
	err := yaml.Unmarshal([]byte(siteConfigStr), &siteConfig)
	assert.NoError(t, err)
	assert.Equal(t,
		siteConfig.Spec.Clusters[0].Proxy,
		Proxy{
			HttpProxy:  "test",
			HttpsProxy: "test",
			NoProxy:    "test",
		},
	)
	assert.Equal(t,
		siteConfig.Spec.Clusters[1].Proxy,
		Proxy{
			HttpProxy:  "",
			HttpsProxy: "",
			NoProxy:    "",
		},
	)
}
