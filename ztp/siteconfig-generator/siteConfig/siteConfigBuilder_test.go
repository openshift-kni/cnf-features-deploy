package siteConfig

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	//"fmt"

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
	scBuilder.setLocalExtraManifestPath("../../source-crs/extra-manifest")
	_, err = scBuilder.Build(sc)
	assert.Equal(t, err, errors.New("Error: Repeated Cluster Name test-site/cluster1"))
}

func Test_siteConfigBuildExtraManifest(t *testing.T) {
	sc := SiteConfig{}
	err := yaml.Unmarshal([]byte(siteConfigTest), &sc)
	assert.Equal(t, err, nil)

	scBuilder, _ := NewSiteConfigBuilder()

	// Expect to fail as the localExtraManifest path is in its default value
	_, err = scBuilder.Build(sc)
	assert.NotEqual(t, err, nil)
	assert.Equal(t, strings.Contains(err.Error(), "no such file or directory"), true)

	// Set the local extra manifest path to cnf-feature-deploy/ztp/source-crs/extra-manifest
	scBuilder.setLocalExtraManifestPath("../../source-crs/extra-manifest")
	// Setting the networkType to its default value
	sc.Spec.Clusters[0].NetworkType = "OVNKubernetes"
	clustersCRs, err := scBuilder.Build(sc)
	assert.Equal(t, err, nil)
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
	assert.NotEqual(t, err, nil)
	assert.Equal(t, strings.Contains(err.Error(), "no such file or directory"), true)

	// Test override pre defined extra manifest
	sc.Spec.Clusters[0].ExtraManifestPath = "testdata/user-extra-manifest/override-extra-manifest"
	sc.Spec.Clusters[0].NetworkType = "OVNKubernetes"
	_, err = scBuilder.Build(sc)
	assert.NotEqual(t, err, nil)
	assert.Equal(t, err, errors.New("Pre-defined extra-manifest cannot be over written 03-sctp-machine-config.yaml"))

	// Setting valid user extra manifest
	sc.Spec.Clusters[0].ExtraManifestPath = "testdata/user-extra-manifest"
	sc.Spec.Clusters[0].NetworkType = "OVNKubernetes"
	clustersCRs, err = scBuilder.Build(sc)
	assert.Equal(t, err, nil)
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
