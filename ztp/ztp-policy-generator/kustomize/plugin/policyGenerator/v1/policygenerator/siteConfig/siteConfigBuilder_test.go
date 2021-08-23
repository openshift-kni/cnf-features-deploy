package siteConfig

import (
	"os"
	"testing"

	"github.com/openshift-kni/cnf-features-deploy/ztp/ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator/utils"
	"github.com/stretchr/testify/assert"
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
