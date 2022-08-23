package siteConfig

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

const siteConfigTest = `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "test-site"
  namespace: "test-site"
spec:
  baseDomain: "example.com"
  pullSecretRef:
    name: "pullSecretName"
  clusterImageSetNameRef: "openshift-v4.8.0"
  sshPublicKey: "ssh-rsa "
  sshPrivateKeySecretRef:
    name: "sshPrvKey"
  clusters:
  - clusterName: "cluster1"
    clusterType: sno
    numMasters: 1
    networkType: "OVNKubernetes"
    clusterLabels:
      group-du-sno: ""
      common: true
      sites : "test-site"
    clusterNetwork:
      - cidr: 10.128.0.0/14
        hostPrefix: 23
    machineNetwork:
      - cidr: 10.16.231.0/24
    serviceNetwork:
      - 172.30.0.0/16
    additionalNTPSources:
      - NTP.server1
      - 10.16.231.22
    nodes:
      - hostName: "node1"
        biosConfigRef:
          filePath: "../../siteconfig-generator-kustomize-plugin/testSiteConfig/testHW.profile"
        bmcAddress: "idrac-virtualmedia+https://1.2.3.4/redfish/v1/Systems/System.Embedded.1"
        bmcCredentialsName:
          name: "name of bmcCredentials secret"
        bootMACAddress: "00:00:00:01:20:30"
        bootMode: "UEFI"
        rootDeviceHints:
          hctl: "1:2:0:0"
        cpuset: "2-19,22-39"
        nodeNetwork:
          interfaces:
            - name: eno1
              macAddress: "00:00:00:01:20:30"
          config:
            interfaces:
              - name: eno1
                macAddress: "00:00:00:01:20:30"
                type: ethernet
                ipv4:
                  enabled: true
                  dhcp: false
        diskPartition:
           - device: /dev/sda
             partitions:
               - mount_point: /var/imageregistry
                 size: 102500
                 start: 344844
`

const siteConfigStandardClusterTest = `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "test-standard"
  namespace: "test-standard"
spec:
  baseDomain: "example.com"
  pullSecretRef:
    name: "pullSecretName"
  clusterImageSetNameRef: "openshift-v4.9.0"
  sshPublicKey: "ssh-rsa "
  clusters:
  - clusterName: "cluster1"
    clusterNetwork:
      - cidr: 10.128.0.0/14
        hostPrefix: 23
    machineNetwork:
      - cidr: 10.16.231.0/24
    serviceNetwork:
      - 172.30.0.0/16
    nodes:
      - hostName: "node1"
        nodeNetwork:
          interfaces:
            - name: eno1
              macAddress: "00:00:00:01:20:30"
      - hostName: "node2"
        nodeNetwork:
          interfaces:
            - name: eno1
              macAddress: "00:00:00:01:20:40"
      - hostName: "node3"
        nodeNetwork:
          interfaces:
            - name: eno1
              macAddress: "00:00:00:01:20:50"
`

func Test_grtManifestFromTemplate(t *testing.T) {
	tests := []struct {
		template      string
		data          interface{}
		expectFn      string
		expectContent string
	}{{
		template:      "testdata/good.yaml.tmpl",
		data:          struct{ Value string }{"values"},
		expectFn:      "role-good.yaml",
		expectContent: "rendered-role: values\n",
	}, {
		template: "testdata/parse_failure.yaml.tmpl",
	}, {
		template: "testdata/execution_failure.yaml.tmpl",
	}, {
		template: "testdata/empty.yaml.tmpl",
	}}
	// Cannot test bad filename because that causes a panic
	scb := SiteConfigBuilder{}
	for _, test := range tests {
		fn, content, _ := scb.getManifestFromTemplate(test.template, "role", test.data)
		assert.Equal(t, test.expectFn, fn)
		assert.Equal(t, test.expectContent, content)
	}
}

// Helper function to find the generated CR of the given kind. Returns nil,error when not found
func getKind(builtCRs []interface{}, kind string) (map[string]interface{}, error) {
	for _, cr := range builtCRs {
		mapSourceCR := cr.(map[string]interface{})
		if mapSourceCR["kind"] == kind {
			return mapSourceCR, nil
		}
	}
	return nil, errors.New("Error: Did not find " + kind + " in result")
}

func Test_siteConfigBuildValidation(t *testing.T) {

	sc := SiteConfig{}
	err := yaml.Unmarshal([]byte(siteConfigTest), &sc)
	assert.Equal(t, err, nil)

	scBuilder, _ := NewSiteConfigBuilder()
	// Set empty cluster Name
	sc.Spec.Clusters[0].ClusterName = ""
	_, err = scBuilder.Build(sc)
	assert.Equal(t, err, errors.New("Error: Missing cluster name at site test-site"))

	// Set invalid network type
	sc.Spec.Clusters[0].ClusterName = "cluster1"
	sc.Spec.Clusters[0].NetworkType = "invalidNetworkType"
	_, err = scBuilder.Build(sc)
	assert.Equal(t, err, errors.New("Error: networkType must be either OpenShiftSDN or OVNKubernetes test-site/cluster1"))

	// Set repeated cluster names
	sc.Spec.Clusters[0].NetworkType = "OVNKubernetes"
	sc.Spec.Clusters = append(sc.Spec.Clusters, sc.Spec.Clusters[0])
	scBuilder.SetLocalExtraManifestPath("testdata/extra-manifest")
	_, err = scBuilder.Build(sc)
	assert.Equal(t, err, errors.New("Error: Repeated Cluster Name test-site/cluster1"))

	// Set empty clusterImageSetnameRef
	sc.Spec.Clusters[0].ClusterName = "cluster1"
	sc.Spec.Clusters[0].NetworkType = "OVNKubernetes"
	sc.Spec.ClusterImageSetNameRef = ""
	sc.Spec.Clusters[0].ClusterImageSetNameRef = ""
	_, err = scBuilder.Build(sc)
	assert.Equal(t, err, errors.New("Error: Site and cluster clusterImageSetNameRef cannot be empty test-site/cluster1"))
}

func Test_siteConfigDifferentClusterVersions(t *testing.T) {
	sc := SiteConfig{}
	err := yaml.Unmarshal([]byte(siteConfigTest), &sc)
	assert.Equal(t, err, nil)
	scBuilder, _ := NewSiteConfigBuilder()
	// Set a site clusterImageSetNameRef and empty cluster clusterImageSetNameRef
	sc.Spec.ClusterImageSetNameRef = "openshift-4.8"
	sc.Spec.Clusters[0].ClusterImageSetNameRef = ""
	_, err = scBuilder.Build(sc)
	// expect cluster's clusterImageSetNameRef to match site's clusterImageSetNameRef
	assert.Equal(t, sc.Spec.Clusters[0].ClusterImageSetNameRef, "openshift-4.8")
	// Setspecific clusterImageSetNameRef for a specific cluster
	sc.Spec.Clusters[0].ClusterImageSetNameRef = "openshift-4.9"
	_, err = scBuilder.Build(sc)
	// expect cluster's clusterImageSetNameRef to be set to the specific release defined in the cluster
	assert.Equal(t, sc.Spec.Clusters[0].ClusterImageSetNameRef, "openshift-4.9")
}

func Test_siteConfigBuildExtraManifestPaths(t *testing.T) {

	sc := SiteConfig{}
	err := yaml.Unmarshal([]byte(siteConfigTest), &sc)
	assert.NoError(t, err)

	relativeManifestPath := "testdata/extra-manifest"
	absoluteManifestPath, err := filepath.Abs(relativeManifestPath)
	assert.Equal(t, err, nil)

	// Test 1: Test with relative manifest path

	scBuilder, _ := NewSiteConfigBuilder()
	scBuilder.SetLocalExtraManifestPath(relativeManifestPath)
	// Setting the networkType to its default value
	sc.Spec.Clusters[0].NetworkType = "OVNKubernetes"
	_, err = scBuilder.Build(sc)
	assert.NoError(t, err)

	// Test 2: Test with absolute manifest path

	scBuilder, _ = NewSiteConfigBuilder()
	scBuilder.SetLocalExtraManifestPath(absoluteManifestPath)
	sc.Spec.Clusters[0].NetworkType = "OVNKubernetes"
	_, err = scBuilder.Build(sc)
	assert.NoError(t, err)
}

func Test_siteConfigBuildExtraManifest(t *testing.T) {
	sc := SiteConfig{}
	err := yaml.Unmarshal([]byte(siteConfigTest), &sc)
	assert.NoError(t, err)

	scBuilder, _ := NewSiteConfigBuilder()

	// Expect to fail as the localExtraManifest path is in its default value
	_, err = scBuilder.Build(sc)
	if assert.Error(t, err) {
		assert.Equal(t, strings.Contains(err.Error(), "no such file or directory"), true)
	}

	// Set the local extra manifest path to cnf-feature-deploy/ztp/source-crs/extra-manifest
	scBuilder.SetLocalExtraManifestPath("testdata/extra-manifest")

	// Setting the networkType to its default value
	sc.Spec.Clusters[0].NetworkType = "OVNKubernetes"
	clustersCRs, err := scBuilder.Build(sc)
	assert.NoError(t, err)
	assert.NotEqual(t, clustersCRs["test-site/cluster1"], nil)
	// expect all the installation CRs generated
	assert.Equal(t, len(clustersCRs["test-site/cluster1"]), len(scBuilder.SourceClusterCRs))
	// check for the workload added
	for _, cr := range clustersCRs["test-site/cluster1"] {
		mapSourceCR := cr.(map[string]interface{})

		if mapSourceCR["kind"] == "ConfigMap" {
			dataMap := mapSourceCR["data"].(map[string]interface{})
			assert.NotEqual(t, dataMap["03-master-workload-partitioning.yaml"], nil)
			break
		}
	}

	// Setting invalid user extra manifest path
	sc.Spec.Clusters[0].ExtraManifestPath = "invalid-path/extra-manifest"
	sc.Spec.Clusters[0].NetworkType = "OVNKubernetes"
	_, err = scBuilder.Build(sc)
	if assert.Error(t, err) {
		assert.Equal(t, strings.Contains(err.Error(), "no such file or directory"), true)
	}

	// Test override pre defined extra manifest
	sc.Spec.Clusters[0].ExtraManifestPath = "testdata/user-extra-manifest/override-extra-manifest"
	sc.Spec.Clusters[0].NetworkType = "OVNKubernetes"
	_, err = scBuilder.Build(sc)
	if assert.Error(t, err) {
		assert.True(t, strings.HasPrefix(err.Error(), "Pre-defined extra-manifest cannot be over written"))
	}

	// Setting valid user extra manifest
	sc.Spec.Clusters[0].ExtraManifestPath = "testdata/user-extra-manifest"
	sc.Spec.Clusters[0].NetworkType = "OVNKubernetes"
	clustersCRs, err = scBuilder.Build(sc)
	assert.NoError(t, err)
	assert.NotEqual(t, clustersCRs["test-site/cluster1"], nil)
	// expect all the installation CRs generated
	assert.Equal(t, len(clustersCRs["test-site/cluster1"]), len(scBuilder.SourceClusterCRs))

	// check for the user extra manifest added
	for _, cr := range clustersCRs["test-site/cluster1"] {
		mapSourceCR := cr.(map[string]interface{})

		if mapSourceCR["kind"] == "ConfigMap" {
			dataMap := mapSourceCR["data"].(map[string]interface{})
			assert.NotNil(t, dataMap["user-extra-manifest.yaml"])
			assert.Nil(t, dataMap[".bad-non-yaml-file.yaml"])
			break
		}
	}
}

func Test_getExtraManifestTemplatedRoles(t *testing.T) {
	sc := SiteConfig{}
	err := yaml.Unmarshal([]byte(siteConfigTest), &sc)
	assert.NoError(t, err)
	cluster := sc.Spec.Clusters[0]

	tests := []struct {
		roles []string
	}{{
		roles: []string{"master"},
	}, {
		roles: []string{"master", "worker"},
	}, {
		roles: []string{"master", "worker", "worker-du", "worker-cu", "etc"},
	}}
	// Cannot test bad filename because that causes a panic
	scb := SiteConfigBuilder{}
	scb.scBuilderExtraManifestPath = "testdata/role-templates"
	for _, test := range tests {
		cluster.Nodes = []Nodes{}
		for _, role := range test.roles {
			cluster.Nodes = append(cluster.Nodes, Nodes{
				HostName: fmt.Sprintf("node-%s", role),
				Role:     role,
			})
		}

		dataMap, err := scb.getExtraManifest(map[string]interface{}{}, cluster)
		assert.NoError(t, err)

		for _, role := range test.roles {
			assert.NotNil(t, dataMap[fmt.Sprintf("%s-good.yaml", role)], "Expected extra-manifests for role %s", role)
		}
	}
}

func Test_mergeExtraManifestsExcludeTemplates(t *testing.T) {
	sc := SiteConfig{}
	err := yaml.Unmarshal([]byte(siteConfigTest), &sc)
	assert.NoError(t, err)

	scb := SiteConfigBuilder{}
	scb.SetLocalExtraManifestPath("testdata/extra-manifest")
	scb.scBuilderExtraManifestPath = "testdata/role-templates"

	cluster := sc.Spec.Clusters[0]
	cluster.Nodes = []Nodes{}
	roles := []string{"master", "worker"}
	for _, role := range roles {
		cluster.Nodes = append(cluster.Nodes, Nodes{
			HostName: fmt.Sprintf("node-%s", role),
			Role:     role,
		})
	}

	dataMap, err := scb.getExtraManifest(map[string]interface{}{}, cluster)
	assert.NoError(t, err)

	for _, role := range roles {
		assert.NotNil(t, dataMap[fmt.Sprintf("%s-good.yaml", role)], "Expected extra-manifests for role %s", role)
	}
}

func Test_getClusterCR(t *testing.T) {
	siteConfigYaml := `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "test-site"
  namespace: "test-site"
spec:
  baseDomain: "example.com"
  clusterImageSetNameRef: "openshift-v4.8.0"
  sshPublicKey:
  clusters:
  - clusterName: "cluster1"
    clusterLabels:
      group-du-sno: ""
      common: true
      sites : "test-site"
    nodes:
      - hostName: "node1"
`
	tests := []struct {
		what           string
		input          string
		nodeId         int
		expectedResult string
		expectedError  string
	}{{
		what:           "No template (string)",
		input:          `key: value`,
		expectedResult: `key: value`,
	}, {
		what:           "No template (int)",
		input:          `key: 42`,
		expectedResult: `key: 42`,
	}, {
		what:           "Simple template: Site",
		input:          `key: "{{ .Site.BaseDomain }}"`,
		expectedResult: `key: "example.com"`,
	}, {
		what:           "Simple template: Cluster",
		input:          `key: "{{ .Cluster.ClusterName }}"`,
		expectedResult: `key: "cluster1"`,
	}, {
		what:           "Simple template: Node",
		input:          `key: "{{ .Node.HostName }}"`,
		expectedResult: `key: "node1"`,
	}, {
		what:           "Empty string value",
		input:          `key: "{{ .Site.SshPublicKey }}"`,
		expectedResult: ``,
	}, {
		what:           "Empty slice value",
		input:          `key: "{{ .Cluster.MachineNetwork }}"`,
		expectedResult: ``,
	}, {
		what:           "Empty struct value",
		input:          `key: "{{ .Site.PullSecretRef }}"`,
		expectedResult: ``,
	}, {
		// TODO: This should be an error, not a success that returns a nil map
		what:           "Invalid Node template (no node ID provided)",
		input:          `key: "{{ .Node.HostName }}"`,
		nodeId:         -1,
		expectedResult: ``,
	}, {
		what:          "Unparsable key",
		input:         `key: "{{ Good luck! }}"`,
		nodeId:        -1,
		expectedError: "could not be translated",
	}, {
		// TODO: This should be an error, not a success that returns a nil map
		what:           "Invalid key (missing field)",
		input:          `key: "{{ .Site.NoSuchKey }}"`,
		expectedResult: ``,
	}, {
		// TODO: This should be an error, not a success that returns a nil map
		what:           "Invalid key (going too deep on a leaf)",
		input:          `key: "{{ .Node.HostName.Too.Deep }}"`,
		expectedResult: ``,
	}, {
		what:           "Nested structure (no templates)",
		input:          `top: { middle: { a: "value", b: 42} }`,
		expectedResult: `top: { middle: { a: "value", b: 42} }`,
	}, {
		what:           "Nested structure with templates",
		input:          `top: { a: "{{ .Site.BaseDomain }}", middle: { b: "{{ .Cluster.ClusterName }}", bottom: "end" } }`,
		expectedResult: `top: { a: "example.com", middle: { b: "cluster1", bottom: "end" } }`,
	}, {
		what:           "Nested structure appending another nested structure",
		input:          `top: { a: "{{ .Cluster.ClusterLabels }}", middle: { bottom: "end" } }`,
		expectedResult: `top: { a: {common: "true", "group-du-sno": "", "sites": "test-site"}, middle: { bottom: "end" } }`,
	}, {
		// TODO: This should be an error, not a success where the key is removed
		what:           "Nested recursive structure with invalid key",
		input:          `top: { middle: { bottom: "{{ .Cluster.NoSuchKey }}" } }`,
		expectedResult: `top: { middle: {} }`,
	}}

	sc := SiteConfig{}
	err := yaml.Unmarshal([]byte(siteConfigYaml), &sc)
	assert.NoError(t, err)

	scBuilder, err := NewSiteConfigBuilder()
	assert.NoError(t, err)

	err = scBuilder.validateSiteConfig(sc)
	assert.NoError(t, err)

	for _, test := range tests {
		var source map[string]interface{}
		err := yaml.Unmarshal([]byte(test.input), &source)
		assert.NoError(t, err, test.what)

		result, err := scBuilder.getClusterCR(0, sc, source, test.nodeId)
		if test.expectedError == "" && assert.NoError(t, err, test.what) {
			// To make sure results and expectations match most robustly,
			// unmarshal the expected result, then re-marshal and compare both
			// the result and the expectation:
			var expectedParsedResult map[string]interface{}
			err := yaml.Unmarshal([]byte(test.expectedResult), &expectedParsedResult)
			assert.NoError(t, err, test.what)
			strExpected, err := yaml.Marshal(expectedParsedResult)
			assert.NoError(t, err, test.what)
			strResult, err := yaml.Marshal(result)
			assert.NoError(t, err, test.what)
			assert.Equal(t, string(strExpected), string(strResult), test.what)
		} else if assert.Error(t, err, test.what) {
			assert.Contains(t, err.Error(), test.expectedError, test.what)
		}
	}
}

func Test_SNOClusterSiteConfigBuild(t *testing.T) {
	sc := SiteConfig{}
	err := yaml.Unmarshal([]byte(siteConfigTest), &sc)
	assert.NoError(t, err)

	outputStr := checkSiteConfigBuild(t, sc)
	filesData, err := ReadFile("testdata/siteConfigTestOutput.yaml")
	assert.Equal(t, string(filesData), outputStr)
}

func Test_StandardClusterSiteConfigBuild(t *testing.T) {
	sc := SiteConfig{}
	err := yaml.Unmarshal([]byte(siteConfigStandardClusterTest), &sc)
	assert.NoError(t, err)

	outputStr := checkSiteConfigBuild(t, sc)
	filesData, err := ReadFile("testdata/siteConfigStandardClusterTestOutput.yaml")
	assert.Equal(t, string(filesData), outputStr)
}

func checkSiteConfigBuild(t *testing.T, sc SiteConfig) string {
	scBuilder, err := NewSiteConfigBuilder()
	scBuilder.SetLocalExtraManifestPath("testdata/extra-manifest")
	clustersCRs, err := scBuilder.Build(sc)
	assert.NoError(t, err)
	assert.Equal(t, len(clustersCRs), len(sc.Spec.Clusters))

	var outputBuffer bytes.Buffer
	for _, clusterCRs := range clustersCRs {
		for _, clusterCR := range clusterCRs {
			cr, err := yaml.Marshal(clusterCR)
			assert.NoError(t, err)

			outputBuffer.Write(Separator)
			outputBuffer.Write(cr)
		}
	}
	str := outputBuffer.String()
	outputBuffer.Reset()
	if str == "" && len(err.Error()) > 0 {
		return err.Error()
	}
	return str
}

func Test_CRTemplateOverride(t *testing.T) {
	tests := []struct {
		what                    string
		siteCrTemplates         map[string]string
		clusterCrTemplates      map[string]string
		nodeCrTemplates         map[string]string
		eachCrTemplates         map[string]string
		expectedErrorContains   string
		expectedSearchCollector bool
		expectedBmhInspection   string
	}{{
		what:                    "No overrides",
		expectedErrorContains:   "",
		expectedSearchCollector: false,
		expectedBmhInspection:   "disabled",
	}, {
		what:                    "Override KlusterletAddonConfig at the site level",
		siteCrTemplates:         map[string]string{"KlusterletAddonConfig": "testdata/KlusterletAddonConfigOverride.yaml"},
		expectedErrorContains:   "",
		expectedSearchCollector: true,
		expectedBmhInspection:   "disabled",
	}, {
		what:                  "Override KlusterletAddonConfig with missing metadata",
		clusterCrTemplates:    map[string]string{"KlusterletAddonConfig": "testdata/KlusterletAddonConfigOverride-MissingMetadata.yaml"},
		expectedErrorContains: "Overriden template metadata in",
	}, {
		what:                  "Override KlusterletAddonConfig with missing metadata.annotations",
		clusterCrTemplates:    map[string]string{"KlusterletAddonConfig": "testdata/KlusterletAddonConfigOverride-MissingMetadataAnnotations.yaml"},
		expectedErrorContains: "Overriden template metadata annotations in",
	}, {
		what:                  "Override KlusterletAddonConfig with missing metadata.annotations.argocd",
		clusterCrTemplates:    map[string]string{"KlusterletAddonConfig": "testdata/KlusterletAddonConfigOverride-MissingArgocdAnnotation.yaml"},
		expectedErrorContains: "does not match expected value",
	}, {
		what:                    "Override KlusterletAddonConfig at the cluster level",
		clusterCrTemplates:      map[string]string{"KlusterletAddonConfig": "testdata/KlusterletAddonConfigOverride.yaml"},
		expectedErrorContains:   "",
		expectedSearchCollector: true,
		expectedBmhInspection:   "disabled",
	}, {
		what:                    "Override KlusterletAddonConfig at the cluster level",
		clusterCrTemplates:      map[string]string{"KlusterletAddonConfig": "testdata/KlusterletAddonConfigOverride-NotTemplated.yaml"},
		expectedErrorContains:   "",
		expectedSearchCollector: true,
		expectedBmhInspection:   "disabled",
	}, {
		what:                  "Override KlusterletAddonConfig at the node level",
		nodeCrTemplates:       map[string]string{"KlusterletAddonConfig": "testdata/KlusterletAddonConfigOverride.yaml"},
		expectedErrorContains: `"KlusterletAddonConfig" is not a valid CR type`,
	}, {
		what:                    "Override BareMetalHost",
		eachCrTemplates:         map[string]string{"BareMetalHost": "testdata/BareMetalHostOverride.yaml"},
		expectedSearchCollector: false,
		expectedErrorContains:   "",
		expectedBmhInspection:   "enabled",
	}, {
		what:                  "Override with a missing file",
		eachCrTemplates:       map[string]string{"BareMetalHost": "no/such/path.yaml"},
		expectedErrorContains: "no/such/path.yaml",
	}, {
		what:                  "Override with an invalid kind",
		eachCrTemplates:       map[string]string{"BlusterletSaddleConfig": "testdata/KlusterletAddonConfigOverride.yaml"},
		expectedErrorContains: `"BlusterletSaddleConfig" is not a valid CR type`,
	}, {
		what:                  "Override with an unparseable yaml file",
		eachCrTemplates:       map[string]string{"BareMetalHost": "testdata/notyaml.yaml"},
		expectedErrorContains: "unmarshal errors",
	}, {
		what:                  "Override with a mismatched yaml file",
		eachCrTemplates:       map[string]string{"BareMetalHost": "testdata/KlusterletAddonConfigOverride.yaml"},
		expectedErrorContains: "does not match expected kind",
	}, {
		what:                  "Override with a mismatched hard-coded metadata.name",
		eachCrTemplates:       map[string]string{"BareMetalHost": "testdata/BareMetalHostOverride-badName.yaml"},
		expectedErrorContains: " metadata.name ",
	}, {
		what:                  "Override with a mismatched hard-coded metadata.namespace",
		eachCrTemplates:       map[string]string{"BareMetalHost": "testdata/BareMetalHostOverride-badNamespace.yaml"},
		expectedErrorContains: " metadata.namespace ",
	}, {
		what:                  "Override with a mismatched hard-coded argocd annotation",
		eachCrTemplates:       map[string]string{"BareMetalHost": "testdata/BareMetalHostOverride-badAnnotation.yaml"},
		expectedErrorContains: ` metadata.annotations["argocd.argoproj.io/sync-wave"]`,
	}}

	scBuilder, err := NewSiteConfigBuilder()
	scBuilder.SetLocalExtraManifestPath("testdata/extra-manifest")
	assert.NoError(t, err)

	for _, test := range tests {
		setups := make(map[string]func(*SiteConfig))
		if len(test.eachCrTemplates) > 0 {
			// Run the same test thrice, expecting identical results for each
			setups["site"] = func(sc *SiteConfig) {
				sc.Spec.CrTemplates = test.eachCrTemplates
			}
			setups["cluster"] = func(sc *SiteConfig) {
				sc.Spec.Clusters[0].CrTemplates = test.eachCrTemplates
			}
			setups["node"] = func(sc *SiteConfig) {
				sc.Spec.Clusters[0].Nodes[0].CrTemplates = test.eachCrTemplates
			}
		} else {
			// Singleton test: prepare exact overrides
			setups[""] = func(sc *SiteConfig) {
				sc.Spec.CrTemplates = test.siteCrTemplates
				sc.Spec.Clusters[0].CrTemplates = test.clusterCrTemplates
				sc.Spec.Clusters[0].Nodes[0].CrTemplates = test.nodeCrTemplates
			}
		}
		for scope, setup := range setups {
			tag := test.what
			if scope != "" {
				tag = fmt.Sprintf("%s at the %s level", test.what, scope)
			}
			sc := SiteConfig{}
			err = yaml.Unmarshal([]byte(siteConfigTest), &sc)
			assert.NoError(t, err, tag)

			setup(&sc)

			result, err := scBuilder.Build(sc)
			if test.expectedErrorContains == "" {
				if assert.NoError(t, err, tag) {
					assertKlusterletSearchCollector(t, result, test.expectedSearchCollector, "cluster1", tag)
					assertBmhInspection(t, result, test.expectedBmhInspection, "cluster1", "node1", tag)
				}
			} else {
				if assert.Error(t, err, tag) {
					assert.Contains(t, err.Error(), test.expectedErrorContains, tag)
				}
			}
		}
	}
}

func assertKlusterletSearchCollector(t *testing.T, builtCRs map[string][]interface{}, expectedSearchCollector bool, clusterName string, tag string) {
	for _, cr := range builtCRs["test-site/cluster1"] {
		mapSourceCR := cr.(map[string]interface{})
		if mapSourceCR["kind"] == "KlusterletAddonConfig" {
			metadata := mapSourceCR["metadata"].(map[string]interface{})
			assert.Equal(t, clusterName, metadata["name"].(string), tag)
			assert.Equal(t, clusterName, metadata["namespace"].(string), tag)
			spec := mapSourceCR["spec"].(map[string]interface{})
			searchCollector := spec["searchCollector"].(map[string]interface{})
			enabled := searchCollector["enabled"].(bool)
			assert.Equal(t, expectedSearchCollector, enabled, tag)
			break
		}
	}
}

func assertBmhInspection(t *testing.T, builtCRs map[string][]interface{}, expectedBmhInspection string, clusterName, nodeName string, tag string) {
	for _, cr := range builtCRs["test-site/cluster1"] {
		mapSourceCR := cr.(map[string]interface{})
		if mapSourceCR["kind"] == "BareMetalHost" {
			metadata := mapSourceCR["metadata"].(map[string]interface{})
			assert.Equal(t, nodeName, metadata["name"].(string), tag)
			assert.Equal(t, clusterName, metadata["namespace"].(string), tag)
			annotations := metadata["annotations"].(map[string]interface{})
			enabled := annotations["inspect.metal3.io"].(string)
			assert.Equal(t, expectedBmhInspection, enabled, tag)
			break
		}
	}
}

func Test_translateTemplateKey(t *testing.T) {
	tests := []struct {
		input          string
		expectedError  string
		expectedResult string
	}{{
		input:          "{{ .Node.Some.Field.Name }}",
		expectedResult: "siteconfig.Spec.Clusters.Nodes.Some.Field.Name",
	}, {
		input:          "{{ .Cluster.Other.Field.Name }}",
		expectedResult: "siteconfig.Spec.Clusters.Other.Field.Name",
	}, {
		input:          "{{ .Site.Name.Of.Field }}",
		expectedResult: "siteconfig.Spec.Name.Of.Field",
	}, {
		input:          "{{.Site.With.No.Space}}",
		expectedResult: "siteconfig.Spec.With.No.Space",
	}, {
		input:          ".Cluster.With.No.Bracket",
		expectedResult: "siteconfig.Spec.Clusters.With.No.Bracket",
	}, {
		input:         "",
		expectedError: "could not be translated",
	}, {
		input:         "{{ }}",
		expectedError: "could not be translated",
	}, {
		input:         "{{ .Nodes.Incorrect.Prefix }}",
		expectedError: "could not be translated",
	}, {
		input:         "siteconfig.Spec.Old.Style.Tag",
		expectedError: "could not be translated",
	}}
	for _, test := range tests {
		result, err := translateTemplateKey(test.input)
		if test.expectedError == "" {
			assert.NoError(t, err, test.input)
			assert.Equal(t, test.expectedResult, result, test.input)
		} else {
			if assert.Error(t, err, test.input) {
				assert.Contains(t, err.Error(), test.expectedError, test.input)
			}
		}
	}
}

func Test_nmstateConfig(t *testing.T) {
	network := `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "test-site"
  namespace: "test-site"
spec:
  baseDomain: "example.com"
  clusterImageSetNameRef: "openshift-v4.8.0"
  sshPublicKey:
  clusters:
  - clusterName: "cluster1"
    clusterLabels:
      group-du-sno: ""
      common: true
      sites : "test-site"
    nodes:
      - hostName: "node1"
        nodeNetwork:
          interfaces:
            - name: "eno1"
              macAddress: E4:43:4B:F6:12:E0
          config:
            interfaces:
            - name: eno1
              type: ethernet
              state: up
`
	sc := SiteConfig{}
	err := yaml.Unmarshal([]byte(network), &sc)
	assert.Equal(t, err, nil)

	scBuilder, _ := NewSiteConfigBuilder()
	scBuilder.SetLocalExtraManifestPath("testdata/extra-manifest")
	// Check good case, network creates NMStateConfig
	result, err := scBuilder.Build(sc)
	nmState, err := getKind(result["test-site/cluster1"], "NMStateConfig")
	assert.NotNil(t, nmState, nil)

	noNetwork := `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "test-site"
  namespace: "test-site"
spec:
  baseDomain: "example.com"
  clusterImageSetNameRef: "openshift-v4.8.0"
  sshPublicKey:
  clusters:
  - clusterName: "cluster1"
    clusterLabels:
      group-du-sno: ""
      common: true
      sites : "test-site"
    nodes:
      - hostName: "node1"
`

	// Set empty case, no network means no NMStateConfig
	err = yaml.Unmarshal([]byte(noNetwork), &sc)
	assert.Equal(t, err, nil)
	result, err = scBuilder.Build(sc)
	nmState, err = getKind(result["test-site/cluster1"], "NMStateConfig")
	assert.Nil(t, nmState, nil)

	emptyNetwork := `
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  name: "test-site"
  namespace: "test-site"
spec:
  baseDomain: "example.com"
  clusterImageSetNameRef: "openshift-v4.8.0"
  sshPublicKey:
  clusters:
  - clusterName: "cluster1"
    clusterLabels:
      group-du-sno: ""
      common: true
      sites : "test-site"
    nodes:
      - hostName: "node1"
        nodeNetwork:
          interfaces: []
          config: {}
`

	// With empty config and interfaces
	err = yaml.Unmarshal([]byte(emptyNetwork), &sc)
	assert.Equal(t, err, nil)
	result, err = scBuilder.Build(sc)
	nmState, err = getKind(result["test-site/cluster1"], "NMStateConfig")
	assert.Nil(t, nmState, nil)
}

func Test_filterExtraManifests(t *testing.T) {

	getMapWithFileNames := func(root string) map[string]interface{} {
		var dataMap = make(map[string]interface{})
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if filepath.Ext(path) != ".yaml" {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			dataMap[info.Name()] = true
			return nil
		})
		if err != nil {
			fmt.Printf(err.Error())
			return nil
		}

		return dataMap
	}

	const filter = `
  inclusionDefault: %s
  exclude: %s 
  include: %s
`

	type args struct {
		dataMap map[string]interface{}
		filter  string
	}
	tests := []struct {
		name       string
		args       args
		want       map[string]interface{}
		wantErrMsg string
		wantErr    bool
	}{
		{
			name:    "remove files from the list",
			wantErr: false,
			args: args{
				dataMap: getMapWithFileNames("../../source-crs/extra-manifest/"),
				filter:  fmt.Sprintf(filter, ``, `[03-sctp-machine-config-worker.yaml, 03-sctp-machine-config-master.yaml]`, ``),
			},
			want: map[string]interface{}{"01-container-mount-ns-and-kubelet-conf-worker.yaml": true,
				"04-accelerated-container-startup-master.yaml":       true,
				"06-kdump-worker.yaml":                               true,
				"01-container-mount-ns-and-kubelet-conf-master.yaml": true,
				"04-accelerated-container-startup-worker.yaml":       true,
				"06-kdump-master.yaml":                               true,
				"03-workload-partitioning.yaml":                      true},
		},
		{
			name:    "exclude all files except 03-sctp-machine-config-worker.yaml",
			wantErr: false,
			args: args{
				dataMap: getMapWithFileNames("../../source-crs/extra-manifest/"),
				filter:  fmt.Sprintf(filter, `exclude`, ``, `[03-workload-partitioning.yaml]`),
			},
			want: map[string]interface{}{"03-workload-partitioning.yaml": true},
		},
		{
			name:    "error when both include and exclude contain a list of files and user in exclude mode",
			wantErr: true,
			args: args{
				dataMap: getMapWithFileNames("../../source-crs/extra-manifest/"),
				filter:  fmt.Sprintf(filter, `exclude`, `[03-workload-partitioning.yaml]`, `[03-workload-partitioning.yaml]`),
			},
			wantErrMsg: "when InclusionDefault is set to exclude, exclude list can not have entries",
		},
		{
			name:    "error when a file is listed under include list but user in include mode",
			wantErr: true,
			args: args{
				dataMap: getMapWithFileNames("../../source-crs/extra-manifest/"),
				filter:  fmt.Sprintf(filter, `include`, ``, `[03-workload-partitioning.yaml]`),
			},
			wantErrMsg: "when InclusionDefault is set to include, include list can not have entries",
		},
		{
			name:    "error when incorrect value is used for inclusionDefault",
			wantErr: true,
			args: args{
				dataMap: getMapWithFileNames("../../source-crs/extra-manifest/"),
				filter:  fmt.Sprintf(filter, `something_random`, `[03-workload-partitioning.yaml]`, ``),
			},
			wantErrMsg: "acceptable values for inclusionDefault are include and exclude. You have entered something_random",
		},
		{
			name:    "error when trying to remove a file that is not in the dir",
			wantErr: true,
			args: args{
				dataMap: getMapWithFileNames("../../source-crs/extra-manifest/"),
				filter:  fmt.Sprintf(filter, `include`, `[03-my-unknown-file.yaml]`, ``),
			},
			wantErrMsg: "Filename 03-my-unknown-file.yaml under exclude array is invalid. Valid files names are:",
		},
		{
			name:    "error when trying to keep a file that is not in the dir",
			wantErr: true,
			args: args{
				dataMap: getMapWithFileNames("../../source-crs/extra-manifest/"),
				filter:  fmt.Sprintf(filter, `exclude`, ``, `[03-my-unknown-file.yaml]`),
			},
			wantErrMsg: "Filename 03-my-unknown-file.yaml under include array is invalid. Valid files names are:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Filter{}
			err := yaml.Unmarshal([]byte(tt.args.filter), &f)
			got, err := filterExtraManifests(tt.args.dataMap, &f)
			if (err != nil) != tt.wantErr {
				t.Errorf("filterExtraManifests() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if !assert.Contains(t, err.Error(), tt.wantErrMsg) {
					t.Errorf("filterExtraManifests() not happypath got = %v, want %v", err.Error(), tt.wantErrMsg)
				}
			} else {
				if !cmp.Equal(got, tt.want) {
					t.Errorf("filterExtraManifests() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func Test_filterExtraManifestsHigherLevel(t *testing.T) {
	filter := `
       inclusionDefault: %s
       exclude: %s
       include: %s
`
	const s = `
spec:
  clusterImageSetNameRef: "openshift-v4.8.0"
  clusters:
  - clusterName: "cluster1"
    extraManifestPath: testdata/filteredoutput/user-extra-manifest/
    extraManifests:
      filter:
        %s
    nodes:
      - hostName: "node1"
        diskPartition:
           - device: /dev/sda
             partitions:
               - mount_point: /var/imageregistry
                 size: 102500
                 start: 344844
`

	type args struct {
		filter string
	}
	tests := []struct {
		name       string
		args       args
		want       string
		wantErrMsg string
		wantErr    bool
	}{
		{
			name:    "remove files from the list, include generated file from .tmpl and user defined CR",
			wantErr: false,
			args: args{
				filter: fmt.Sprintf(filter, ``, `[user-extra-manifest.yaml, master-image-registry-partition-mc.yaml]`, ``),
			},
			want: "testdata/filteredoutput/partialfilter.yaml",
		},
		{
			name:    "remove all files",
			wantErr: false,
			args: args{
				filter: fmt.Sprintf(filter, `exclude`, ``, ``),
			},
			want: "testdata/filteredoutput/removeAll.yaml",
		},
		{
			name:    "remove except user provided CR",
			wantErr: false,
			args: args{
				filter: fmt.Sprintf(filter, `exclude`, ``, `[user-extra-manifest.yaml]`),
			},
			want: "testdata/filteredoutput/onlyUserCR.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := SiteConfig{}
			scString := fmt.Sprintf(s, tt.args.filter)

			err := yaml.Unmarshal([]byte(scString), &sc)
			if !cmp.Equal(err, nil) {
				t.Errorf("filterExtraManifestsHigherLevel() unmarshall err got = %v, want %v", err.Error(), "no error")
				t.FailNow()
			}

			outputStr := checkSiteConfigBuild(t, sc) // TODO improve this method this method
			filesData, err := ReadFile(tt.want)
			if tt.wantErr {
				if !cmp.Equal(outputStr, tt.wantErrMsg) {
					t.Errorf("filterExtraManifestsHigherLevel() error case got = %v, want %v", outputStr, tt.wantErrMsg)
				}
			} else {
				if !cmp.Equal(outputStr, string(filesData)) {
					t.Errorf("filterExtraManifestsHigherLevel() got = %v, want %v", outputStr, string(filesData))
				}
			}
		})
	}

}
