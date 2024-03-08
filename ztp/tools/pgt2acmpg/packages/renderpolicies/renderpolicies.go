package renderpolicies

import (
	"fmt"
	"os"

	"github.com/openshift-kni/cnf-features-deploy/ztp/tools/pgt2acmpg/packages/fileutils"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/types"
)

const (
	PgtRenderedYAMLFileName   = "pgt-out.yaml"
	AcmPGRenderedYAMLFileName = "acmpg-out.yaml"
)

// RenderTemplatePolicy Render the individual ACM policies defined by a template
func RenderTemplatePolicy(templatePath string) (policyYAML []byte, err error) {
	options := krusty.MakeDefaultOptions()
	options.LoadRestrictions = types.LoadRestrictionsNone
	options.PluginConfig = types.EnabledPluginConfig(types.BploLoadFromFileSys)
	k := krusty.MakeKustomizer(options)
	resMap, err := k.Run(filesys.MakeFsOnDisk(), templatePath)
	if err != nil {
		return policyYAML, fmt.Errorf("failed to Kustomize template: %w", err)
	}
	policyYAML, err = resMap.AsYaml()
	if err != nil {
		return policyYAML, fmt.Errorf("failed to convert the patched manifest(s) back to YAML: %w", err)
	}
	return policyYAML, nil
}

// RenderAndWriteTemplateToYAML Render the individual ACM policies defined by a template to a file
func RenderAndWriteTemplateToYAML(templatePath, outputFile string) (err error) {
	var policyYAML []byte
	policyYAML, err = RenderTemplatePolicy(templatePath)
	if err != nil {
		return fmt.Errorf("error rendering template to file, err: %s", err)
	}
	err = os.WriteFile(outputFile, policyYAML, fileutils.DefaultFileWritePermissions)
	if err != nil {
		return fmt.Errorf("error writing to file, err: %s", err)
	}
	return nil
}
