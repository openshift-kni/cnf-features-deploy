// Copyright Contributors to the Open Cluster Management project
// The code in this file is based on https://github.com/open-cluster-management-io/policy-generator-plugin/blob/main/internal/patches.go

package patches

import (
	"errors"
	"fmt"
	"os"
	"path"

	yaml "gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type ManifestPatcher struct {
	// The manifests to patch.
	Manifests []map[string]interface{}
	// The Kustomize patches to apply on the manifests. Note that modifications are made
	// to the input maps. If this is an issue, provide a deep copy of the patches.
	Patches []map[string]interface{}

	OpenAPI []map[string]interface{}
}

// validateManifestInfo verifies that the apiVersion, kind, metadata.name fields from a manifest
// are set. If at least one is not present, an error is returned based on the input error template
// which accepts the field name.
func validateManifestInfo(manifest map[string]interface{}, errTemplate string) error {
	apiVersion, _, _ := unstructured.NestedString(manifest, "apiVersion")
	if apiVersion == "" {
		return fmt.Errorf(errTemplate, "apiVersion")
	}

	kind, _, _ := unstructured.NestedString(manifest, "kind")
	if kind == "" {
		return fmt.Errorf(errTemplate, "kind")
	}

	name, _, _ := unstructured.NestedString(manifest, "metadata", "name")
	if name == "" {
		return fmt.Errorf(errTemplate, "metadata.name")
	}

	return nil
}

// Validate performs basic validatation of the manifests and patches. Any missing values in the
// patches that can be derived are added to the patches. An error is returned if a manifest or
// patch is invalid.
func (m *ManifestPatcher) Validate() error {
	if len(m.Manifests) == 0 {
		return errors.New("there must be one or more manifests")
	}

	// Validate the manifest fields for applying patches
	const errTemplate = `all manifests must have the "%s" field set to a non-empty string`
	for _, manifest := range m.Manifests {
		err := validateManifestInfo(manifest, errTemplate)
		if err != nil {
			return err
		}
	}

	// If there is more than a single manifest, the patch must contain a name, kind, and apiVersion.
	if len(m.Manifests) > 1 {
		const patchErrTemplate = `patches must have the "%s" field set to a non-empty string ` +
			`when there is more than one manifest it can apply to`

		for _, patch := range m.Patches {
			err := validateManifestInfo(patch, patchErrTemplate)
			if err != nil {
				return err
			}
		}

		// At this point, there is a reasonable chance that the patch is valid. Kustomize can handle
		// further validation.
		return nil
	}

	// At this point, we know we are only dealing with a single manifest, so we can assume all
	// the patches are meant to apply to it.
	manifest := (m.Manifests)[0]
	// The following fields have already been confirmed to exist and be valid previously in this
	// method.
	apiVersion, _, _ := unstructured.NestedString(manifest, "apiVersion")
	kind, _, _ := unstructured.NestedString(manifest, "kind")
	name, _, _ := unstructured.NestedString(manifest, "metadata", "name")
	// The namespace would only apply to a manifest for a namespaced resource, so this is optional.
	// Treat an empty string as meaning the manifest is for a cluster-wide resource.
	namespace, _, _ := unstructured.NestedString(manifest, "metadata", "namespace")

	// Apply defaults on the patches
	for i := range m.Patches {
		err := setPatchDefaults(apiVersion, kind, name, namespace, m.Patches[i])
		if err != nil {
			return err
		}
	}

	return nil
}

// setPatchDefaults any missing values on the patches that can be retrieved from already known values.
// An error is returned if a patch is in an invalid format.
func setPatchDefaults(
	apiVersion, kind, name, namespace string, patch map[string]interface{},
) error {
	errTemplate := `failed to retrieve the "%s" field from the manifest of name "` + name +
		`"` + ` and kind "` + kind + `": %v`
	setErrTemplate := `failed to set the "%s" field on the patch from the manifest of name "` +
		name + `"` + ` and kind "` + kind + `": %v`

	patchAPIVersion, _, err := unstructured.NestedString(patch, "apiVersion")
	if err != nil {
		return fmt.Errorf(errTemplate, "apiVersion", err)
	}

	if patchAPIVersion == "" {
		err = unstructured.SetNestedField(patch, apiVersion, "apiVersion")
		if err != nil {
			return fmt.Errorf(setErrTemplate, "apiVersion", err)
		}
	}

	patchKind, _, err := unstructured.NestedString(patch, "kind")
	if err != nil {
		return fmt.Errorf(errTemplate, "kind", err)
	}

	if patchKind == "" {
		err = unstructured.SetNestedField(patch, kind, "kind")
		if err != nil {
			return fmt.Errorf(setErrTemplate, "kind", err)
		}
	}

	patchName, _, err := unstructured.NestedString(patch, "metadata", "name")
	if err != nil {
		return fmt.Errorf(errTemplate, "metadata.name", err)
	}

	if patchName == "" {
		err = unstructured.SetNestedField(patch, name, "metadata", "name")
		if err != nil {
			return fmt.Errorf(setErrTemplate, "metadata.name", err)
		}
	}

	patchNamespace, _, err := unstructured.NestedString(patch, "metadata", "namespace")
	if err != nil {
		return fmt.Errorf(errTemplate, "metadata.namespace", err)
	}

	if patchNamespace == "" {
		err = unstructured.SetNestedField(patch, namespace, "metadata", "namespace")
		if err != nil {
			return fmt.Errorf(setErrTemplate, "metadata.namespace", err)
		}
	}

	return nil
}

type KustomizeJSON struct {
	Openapi   `yaml:"openapi"`
	Patches   []Patch  `yaml:"patches"`
	Resources []string `yaml:"resources"`
}
type Openapi struct {
	Path string `yaml:"path"`
}

type Patch struct {
	Path string `yaml:"path"`
}

type Resources []string

// ApplyPatches applies the Kustomize patches on the input manifests using Kustomize and returns
// the patched manifests. An error is returned if the patches can't be applied. This should be
// run after the Validate method.
func (m *ManifestPatcher) ApplyPatches(schema string) ([]map[string]interface{}, error) {
	// Create the file system in memory with the Kustomize YAML files
	fSys := filesys.MakeFsInMemory()

	err := InitializeInMemoryKustomizeDir(fSys, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kustomize dir, err: %s", err)
	}
	kustomizationYAMLFile := KustomizeJSON{}
	kustomizationYAMLFile.Openapi.Path = localSchemaFileName

	options := []struct {
		optionType   string
		kustomizeKey string
		objects      []map[string]interface{}
	}{
		{"manifest", "resources", m.Manifests},
		{"patch", "patches", m.Patches},
	}
	for _, option := range options {
		for i, object := range option.objects {
			var objectYAML []byte
			objectYAML, err := yaml.Marshal(object)
			const errTemplate = "an unexpected error occurred when converting the %s back to " +
				"YAML: %w"

			if err != nil {
				return nil, fmt.Errorf(errTemplate, option.optionType, err)
			}

			manifestFileName := fmt.Sprintf("%s%d.yaml", option.optionType, i)

			err = fSys.WriteFile(path.Join(kustomizeDir, manifestFileName), objectYAML)
			if err != nil {
				return nil, fmt.Errorf(errTemplate, option.optionType, err)
			}

			if option.kustomizeKey == "patches" {
				kustomizationYAMLFile.Patches = append(
					kustomizationYAMLFile.Patches, Patch{Path: manifestFileName})
			}
			if option.kustomizeKey == "resources" {
				kustomizationYAMLFile.Resources = append(
					kustomizationYAMLFile.Resources, manifestFileName)
			}
		}
	}

	return KustomizeManifest(fSys, &kustomizationYAMLFile)
}

const (
	localSchemaFileName = "schema.json"
	kustomizeDir        = "kustomize"
)

// Initializes the in-memory file system with base directory and open API schema
func InitializeInMemoryKustomizeDir(fSys filesys.FileSystem, schema string) (err error) {
	err = fSys.Mkdir(kustomizeDir)
	if err != nil {
		return fmt.Errorf("an unexpected error occurred when configuring Kustomize: %w", err)
	}
	schemaJSON, err := os.ReadFile(schema)
	if err != nil {
		return fmt.Errorf("unable to open file: %s, err: %s ", schema, err)
	}

	err = fSys.WriteFile(path.Join(kustomizeDir, localSchemaFileName), schemaJSON)
	if err != nil {
		return fmt.Errorf("error writing schema, err:%s", err)
	}
	return nil
}

// Runs Kustomize to apply patches in the in-memory file system
func KustomizeManifest(fSys filesys.FileSystem, kustomizationYAMLFile *KustomizeJSON) (manifests []map[string]interface{}, err error) {
	var kustomizationYAML []byte
	kustomizationYAML, err = yaml.Marshal(kustomizationYAMLFile)
	const errTemplate = "an unexpected error occurred when creating the kustomization.yaml file: %w"

	if err != nil {
		return manifests, fmt.Errorf(errTemplate, err)
	}

	err = fSys.WriteFile(path.Join(kustomizeDir, "kustomization.yaml"), kustomizationYAML)
	if err != nil {
		return manifests, fmt.Errorf(errTemplate, err)
	}

	k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())

	resMap, err := k.Run(fSys, "kustomize")
	if err != nil {
		return manifests, fmt.Errorf("failed to apply the patch(es) to the manifest(s) using Kustomize: %w", err)
	}

	manifestsYAML, err := resMap.AsYaml()
	if err != nil {
		return manifests, fmt.Errorf("failed to convert the patched manifest(s) back to YAML: %w", err)
	}

	manifests, err = unmarshalManifestBytes(manifestsYAML)
	if err != nil {
		return manifests, fmt.Errorf("failed to read the patched manifest(s): %w", err)
	}
	return manifests, nil
}
