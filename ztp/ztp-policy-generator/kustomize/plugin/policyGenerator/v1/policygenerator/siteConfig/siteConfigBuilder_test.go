package siteConfig

import (
	"errors"
	"os"
	"testing"

	"github.com/openshift-kni/cnf-features-deploy/ztp/ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator/utils"
	"github.com/stretchr/testify/assert"

	"gopkg.in/yaml.v3"
)

func fh() *utils.FilesHandler {
	d, _ := os.Getwd()
	d = d + "/testdata"
	return utils.NewFilesHandler(d, d, d)
}

func Test_grtManifestFromTemplate(t *testing.T) {
	tests := []struct {
		template      string
		data          interface{}
		expectFn      string
		expectContent string
	}{{
		template:      "good.yaml.tmpl",
		data:          struct{ Value string }{"values"},
		expectFn:      "role-good.yaml",
		expectContent: "rendered-role: values\n",
	}, {
		template: "parse_failure.yaml.tmpl",
	}, {
		template: "execution_failure.yaml.tmpl",
	}, {
		template: "empty.yaml.tmpl",
	}}
	// Cannot test bad filename because that causes a panic
	scb := SiteConfigBuilder{fHandler: fh()}
	for _, test := range tests {
		fn, content, _ := scb.getManifestFromTemplate(test.template, "role", test.data)
		assert.Equal(t, test.expectFn, fn)
		assert.Equal(t, test.expectContent, content)
	}
}

func getKind(builtCRs []interface{}, kind string) (map[string]interface{}, error) {
	for _, cr := range builtCRs {
		mapSourceCR := cr.(map[string]interface{})
		if mapSourceCR["kind"] == kind {
			return mapSourceCR, nil
		}
	}
	return nil, errors.New("Error: Did not find " + kind + " in result")
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

	fh := fh()
	fh.SetResourceBaseDir("../")
	scBuilder, _ := NewSiteConfigBuilder(fh)
	//scBuilder.SetLocalExtraManifestPath("../../source-crs/extra-manifest")
	// Check good case, network creates NMStateConfig
	result, err := scBuilder.Build(sc)
	nmState, err := getKind(result["customResource/test-site/cluster1"], "NMStateConfig")
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
	scBuilder, _ = NewSiteConfigBuilder(fh)
	result, err = scBuilder.Build(sc)
	nmState, err = getKind(result["customResource/test-site/cluster1"], "NMStateConfig")
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
	scBuilder, _ = NewSiteConfigBuilder(fh)
	result, err = scBuilder.Build(sc)
	nmState, err = getKind(result["customResource/test-site/cluster1"], "NMStateConfig")
	assert.Nil(t, nmState, nil)
}
