package siteConfig

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

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
	scBuilder.SetLocalExtraManifestPath("../../source-crs/extra-manifest")
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

func Test_siteConfigBuildExtraManifest(t *testing.T) {
	sc := SiteConfig{}
	err := yaml.Unmarshal([]byte(siteConfigTest), &sc)
	assert.NoError(t, err)

	scBuilder, _ := NewSiteConfigBuilder()

	// Expect to fail as the localExtraManifest path is in its default value
	_, err = scBuilder.Build(sc)
	assert.Error(t, err)
	assert.Equal(t, strings.Contains(err.Error(), "no such file or directory"), true)

	// Set the local extra manifest path to cnf-feature-deploy/ztp/source-crs/extra-manifest
	scBuilder.SetLocalExtraManifestPath("../../source-crs/extra-manifest")
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
			assert.NotEqual(t, dataMap["03-workload-partitioning.yaml"], nil)
			break
		}
	}

	// Setting invalid user extra manifest path
	sc.Spec.Clusters[0].ExtraManifestPath = "invalid-path/extra-manifest"
	sc.Spec.Clusters[0].NetworkType = "OVNKubernetes"
	_, err = scBuilder.Build(sc)
	assert.Error(t, err)
	assert.Equal(t, strings.Contains(err.Error(), "no such file or directory"), true)

	// Test override pre defined extra manifest
	sc.Spec.Clusters[0].ExtraManifestPath = "testdata/user-extra-manifest/override-extra-manifest"
	sc.Spec.Clusters[0].NetworkType = "OVNKubernetes"
	_, err = scBuilder.Build(sc)
	assert.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "Pre-defined extra-manifest cannot be over written"))

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

func Test_getClusterCR(t *testing.T) {
	sc := SiteConfig{}
	err := yaml.Unmarshal([]byte(siteConfigTest), &sc)
	assert.NoError(t, err)

	scBuilder, err := NewSiteConfigBuilder()
	assert.NoError(t, err)

	err = scBuilder.validateSiteConfig(sc)
	assert.NoError(t, err)

	filesData, err := ReadFile("testdata/siteConfigTestOutput.yaml")
	assert.NoError(t, err)

	output := string(filesData)
	for _, sourceCR := range scBuilder.SourceClusterCRs {
		mapSourceCR := sourceCR.(map[string]interface{})
		// Ignore ConfigMap extra-manifest as it has another unit test
		if mapSourceCR["kind"] != "ConfigMap" {
			cr, err := scBuilder.getClusterCR(0, sc, mapSourceCR, 0)
			assert.NoError(t, err)

			crdata, err := yaml.Marshal(cr)
			assert.NoError(t, err)
			// Print the crdata if it fail.
			assert.Equal(t, true, strings.Contains(output, string(crdata)), string(crdata))
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
	scBuilder.SetLocalExtraManifestPath("../../source-crs/extra-manifest")
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
	}}

	scBuilder, err := NewSiteConfigBuilder()
	scBuilder.SetLocalExtraManifestPath("../../source-crs/extra-manifest")
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
