package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"flag"

	"github.com/test-network-function/pgt2acm/packages/acmformat"
	"github.com/test-network-function/pgt2acm/packages/fileutils"
	"github.com/test-network-function/pgt2acm/packages/labels"
	"github.com/test-network-function/pgt2acm/packages/patches"
	"github.com/test-network-function/pgt2acm/packages/pgtformat"
	"github.com/test-network-function/pgt2acm/packages/placement"
	"github.com/test-network-function/pgt2acm/packages/renderpolicies"
	"github.com/test-network-function/pgt2acm/packages/stringhelper"
	"gopkg.in/yaml.v3"
)

func processFlags(inputFile, outputDir, preRenderPatchKindString, sourceCRListString *string) (preRenderPatchKindList, preRenderSourceCRList []string) {
	// Parsing inputs
	flag.Parse()
	if inputFile == nil ||
		outputDir == nil || *inputFile == "" || *outputDir == "" {
		flag.Usage()
		os.Exit(1)
	}
	if preRenderPatchKindString != nil {
		preRenderPatchKindList = strings.Split(*preRenderPatchKindString, ",")
	}
	if sourceCRListString != nil {
		preRenderSourceCRList = strings.Split(*sourceCRListString, ",")
	}
	return preRenderPatchKindList, preRenderSourceCRList
}

func main() {
	// Defines the input PGT directory or file
	var inputFile = flag.String("i", "", "the PGT input file")
	// Defines the output directory for generated ACM templates
	var outputDir = flag.String("o", "", "the ACMGen output Directory")
	// Defines the input schema file. Schema allows patching CRDs containing lists of objects
	var schema = flag.String("s", "", "the optional schema for all non base CRDs")
	// Defines list of manifest kinds to which to pre-render patches to
	var preRenderPatchKindString = flag.String("k", "", "the optional list of manifest kinds for which to pre-render patches")
	// Optionally generates ACM policies for PGT and ACM Gen templates
	var generateACMPolicies = flag.Bool("g", false, "optionally generates ACM policies for PGT and ACM Gen templates")
	// Defines ns.yaml file for templates
	var NSYAML = flag.String("n", fileutils.NamespaceFileName, "the optional ns.yaml file path")
	// optionally disables generating default placement in ns.yaml
	var skipDefaultPlacementBindings = flag.Bool("p", false, "optionally disable generating default placement bindings in ns.yaml")
	// Defines source-crs directory location
	var sourceCRs = flag.String("c", "", "the optional comma delimited list of reference source CRs templates")
	// Optionally generate placement API template containing toleration for
	//     - effect: NoSelect
	//       key: cluster.open-cluster-management.io/unreachable
	var workaroundPlacement = flag.Bool("w", false, "Optional workaround to generate placement API template containing cluster.open-cluster-management.io/unreachable toleration")

	preRenderPatchKindList, preRenderSourceCRList := processFlags(inputFile, outputDir, preRenderPatchKindString, sourceCRs)

	var err error
	if sourceCRs != nil && *sourceCRs != "" {
		err = fileutils.CopySourceCrs(*inputFile, *outputDir, preRenderSourceCRList)
		if err != nil {
			fmt.Printf("Could not copy source-crs files, err: %s", err)
			os.Exit(1)
		}
	}

	allFilesInInputPath, err := fileutils.GetAllYAMLFilesInPath(*inputFile)
	if err != nil {
		fmt.Printf("Could not get file list, err: %s", err)
		os.Exit(1)
	}
	// convert all PGT files
	err = convertAllPGTFiles(preRenderPatchKindList, allFilesInInputPath, inputFile, outputDir, schema, workaroundPlacement)
	if err != nil {
		fmt.Printf("Could not convert PGT files, err: %s", err)
		os.Exit(1)
	}

	if NSYAML != nil && *NSYAML != "" && skipDefaultPlacementBindings != nil {
		err = fileutils.CopyAndProcessNSAndKustomizationYAML(*NSYAML, *inputFile, *outputDir, *skipDefaultPlacementBindings)
		if err != nil {
			fmt.Printf("Could not post-process %s and %s files, err: %s", *NSYAML, fileutils.KustomizationFileName, err)
		}
	}

	if generateACMPolicies != nil && *generateACMPolicies {
		err = renderpolicies.RenderAndWriteTemplateToYAML(*outputDir, renderpolicies.AcmGenRenderedYAMLFileName)
		if err != nil {
			fmt.Printf("Could not generate ACMGen policies, err: %s", err)
			os.Exit(1)
		}

		err = renderpolicies.RenderAndWriteTemplateToYAML(*inputFile, renderpolicies.PgtRenderedYAMLFileName)
		if err != nil {
			fmt.Printf("Could generate PGT policies, err: %s", err)
			os.Exit(1)
		}
	}
}

func convertAllPGTFiles(preRenderPatchKindList, allFilesInInputPath []string, inputFile, outputDir, schema *string, workaroundPlacement *bool) (err error) {
	for _, file := range allFilesInInputPath {
		var kindType fileutils.KindType
		kindType, err = fileutils.GetManifestKind(file)
		if err != nil {
			return fmt.Errorf("could not get manifest kind for file:%s, err: %s", file, err)
		}
		if kindType.Kind != "PolicyGenTemplate" {
			continue
		}
		// Get the relative path
		var relativePath string
		relativePath, err = filepath.Rel(*inputFile, file)
		if err != nil {
			return fmt.Errorf("error getting relative path, err:%s", err)
		}
		err = convertPGTtoACM(*outputDir, file, filepath.Join(*outputDir, fileutils.PrefixLastPathComponent(relativePath, fileutils.ACMPrefix)), *schema, preRenderPatchKindList, workaroundPlacement)
		if err != nil {
			return fmt.Errorf("failed to convert PGT to ACMGen, err=%s", err)
		}
	}
	return nil
}

// Converts an PGT file to a ACM Gen Template file
func convertPGTtoACM(outputDir, inputFile, outputFile, schema string, preRenderPatchKindList []string, workaroundPlacement *bool) (err error) {
	policyGenFileContent, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("unable to open file: %s, err: %s ", inputFile, err)
	}
	policyGenTemp := pgtformat.PolicyGenTemplate{}

	err = yaml.Unmarshal(policyGenFileContent, &policyGenTemp)
	if err != nil {
		return fmt.Errorf("could not unmarshal PolicyGenTemplate data from %s: %s", inputFile, err)
	}

	rootName := policyGenTemp.Metadata.Name
	acmGenTempConversion := acmformat.AcmGenTemplate{}

	seenPoliciesMap := map[string]bool{}
	for srcFileIndex := range policyGenTemp.Spec.SourceFiles {
		seenPoliciesMap[policyGenTemp.Spec.SourceFiles[srcFileIndex].PolicyName] = true
	}

	var seenPoliciesSorted []string
	for policyName := range seenPoliciesMap {
		seenPoliciesSorted = append(seenPoliciesSorted, policyName)
	}

	sort.Strings(seenPoliciesSorted)
	for _, policyName := range seenPoliciesSorted {
		newPolicy := convertPGTPolicyToACMGenPolicy(&policyGenTemp, rootName, policyName, outputDir)
		acmGenTempConversion.Policies = append(acmGenTempConversion.Policies, newPolicy)
		var labelSelector map[string]interface{}
		labelSelector, err = labels.OutputGeneric(labels.LabelToSelector(policyGenTemp.Spec.BindingRules,
			policyGenTemp.Spec.BindingExcludedRules))
		if err != nil {
			return err
		}

		// Convert Miscelanous fields
		convertSimpleMiscellaneousFields(&policyGenTemp, &acmGenTempConversion, rootName)

		if workaroundPlacement != nil && !*workaroundPlacement {
			acmGenTempConversion.PolicyDefaults.Placement.LabelSelector = labelSelector
		} else {
			// starts creating child policies as soon as the managed cluster starts installing
			var placementFilepathRelative string
			placementFilepathRelative, err = placement.GeneratePlacementFile(newPolicy.Name, acmGenTempConversion.PolicyDefaults.Namespace, outputDir, labelSelector)
			if err != nil {
				return fmt.Errorf("error when generating placement file, err: %s", err)
			}
			acmGenTempConversion.PolicyDefaults.Placement.PlacementPath = placementFilepathRelative
		}

		// Apply patches on ACMGen since it is not yet supported officially
		if len(acmGenTempConversion.Policies) > 0 {
			for policyIndex := range acmGenTempConversion.Policies {
				for manifestIndex := range acmGenTempConversion.Policies[policyIndex].Manifests {
					err = RenderPatchesInManifestForSpecifiedKinds(&policyGenTemp, &acmGenTempConversion, policyIndex, manifestIndex, outputDir, schema, preRenderPatchKindList)
					if err != nil {
						return fmt.Errorf("could not render patches in manifest, err:%s", err)
					}
				}
			}
		}
	}
	return writeConvertedTemplateToFile(&policyGenTemp, &acmGenTempConversion, outputFile)
}

func writeConvertedTemplateToFile(policyGenTemp *pgtformat.PolicyGenTemplate, acmGenTempConversion *acmformat.AcmGenTemplate, outputFile string) (err error) {
	convertedContent, err := yaml.Marshal(acmGenTempConversion)
	if err != nil {
		return fmt.Errorf("could not marshall acm profile, err: %s", err)
	}

	convertedContent = []byte("---\n" + string(convertedContent))
	convertedContent = []byte(strings.ReplaceAll(string(convertedContent), "$mcp", policyGenTemp.Spec.Mcp))

	// Ensure the directory exists
	dir := filepath.Dir(outputFile)
	err = os.MkdirAll(dir, fileutils.DefaultDirWritePermissions)
	if err != nil {
		return err
	}

	err = os.WriteFile(outputFile, convertedContent, fileutils.DefaultFileWritePermissions)
	if err != nil {
		fmt.Printf("Error writing to file, err: %s", err)
		return err
	}
	fmt.Printf("Wrote converted ACM template: %s\n", outputFile)
	return nil
}

// Renders patches in manifest
func RenderPatchesInManifestForSpecifiedKinds(policyGenTemp *pgtformat.PolicyGenTemplate, acmGenTempConversion *acmformat.AcmGenTemplate, policyIndex, manifestIndex int, outputDir, schema string, kindsToRender []string) error {
	pathRelativeToOutputDir := filepath.Join(outputDir, acmGenTempConversion.Policies[policyIndex].Manifests[manifestIndex].Path)

	renamedpathRelativeToOutputDir, err := fileutils.RenderMCPLines(pathRelativeToOutputDir, policyGenTemp.Spec.Mcp)
	if err != nil {
		return fmt.Errorf("cannot render MCP lines, err:%s", err)
	}
	relativeManifestPath, err := filepath.Rel(outputDir, renamedpathRelativeToOutputDir)
	if err != nil {
		return fmt.Errorf("cannot get the relative path from path: %s and directory: %s, err:%s", renamedpathRelativeToOutputDir, outputDir, err)
	}

	// we switch to using the renamed manifest file with MCP line commented out
	acmGenTempConversion.Policies[policyIndex].Manifests[manifestIndex].Path = relativeManifestPath
	pathRelativeToOutputDir = renamedpathRelativeToOutputDir

	// Unmarshal the manifest in order to check for metadata patch replacement
	manifestFile, err := patches.UnmarshalManifestFile(pathRelativeToOutputDir)
	if err != nil {
		return fmt.Errorf("could not unmarshall manifest: %s, err: %s", pathRelativeToOutputDir, err)
	}

	if len(manifestFile) == 0 {
		return fmt.Errorf("found empty YAML in the manifest at %s", pathRelativeToOutputDir)
	}

	kind, err := fileutils.GetManifestKind(pathRelativeToOutputDir)
	if err != nil {
		return fmt.Errorf("could not get manifest kind for file: %s, err: %s", pathRelativeToOutputDir, err)
	}

	if !stringhelper.StringInSlice[string](kindsToRender, kind.Kind, false) {
		return nil
	}

	// Patch files only if needed
	if len(acmGenTempConversion.Policies[policyIndex].Manifests[manifestIndex].Patches) == 0 {
		return nil
	}

	patcher := patches.ManifestPatcher{Manifests: manifestFile, Patches: acmGenTempConversion.Policies[policyIndex].Manifests[manifestIndex].Patches}
	const errTemplate = `failed to process the manifest at "%s": %w`

	err = patcher.Validate()
	if err != nil {
		return fmt.Errorf(errTemplate, pathRelativeToOutputDir, err)
	}

	patchedFiles, err := patcher.ApplyPatches(schema)
	if err != nil {
		return fmt.Errorf(errTemplate, pathRelativeToOutputDir, err)
	}
	delete(patchedFiles[0], "apiVersion")
	delete(patchedFiles[0], "kind")

	acmGenTempConversion.Policies[policyIndex].Manifests[manifestIndex].Patches = patchedFiles
	return nil
}

const (
	waveAnnotationKey = "ran.openshift.io/ztp-deploy-wave"
	sourceCrPrefix    = "source-crs"
)

// Converts PGT policy to ACM Gen policy
func convertPGTPolicyToACMGenPolicy(policyGenTemp *pgtformat.PolicyGenTemplate, rootName, policyName, outputDir string) (newPolicy acmformat.PolicyConfig) {
	newPolicy.Name = rootName + "-" + policyName
	newPolicy.PolicyAnnotations = make(map[string]string)
	wave := ""
	for srcFileIndex := range policyGenTemp.Spec.SourceFiles {
		if policyGenTemp.Spec.SourceFiles[srcFileIndex].PolicyName != policyName {
			continue
		}
		newManifest := acmformat.Manifest{Path: sourceCrPrefix + "/" + policyGenTemp.Spec.SourceFiles[srcFileIndex].FileName}

		// Setting EvaluationInterval
		if policyGenTemp.Spec.SourceFiles[srcFileIndex].EvaluationInterval.Compliant != pgtformat.UnsetStringValue {
			newPolicy.EvaluationInterval.Compliant = policyGenTemp.Spec.SourceFiles[srcFileIndex].EvaluationInterval.Compliant
		}

		if policyGenTemp.Spec.SourceFiles[srcFileIndex].EvaluationInterval.NonCompliant != pgtformat.UnsetStringValue {
			newPolicy.EvaluationInterval.NonCompliant = policyGenTemp.Spec.SourceFiles[srcFileIndex].EvaluationInterval.NonCompliant
		}

		newPatch := make(map[string]interface{})
		hasPatch := false
		if len(policyGenTemp.Spec.SourceFiles[srcFileIndex].Metadata) != 0 {
			hasPatch = true
			newPatch["metadata"] = policyGenTemp.Spec.SourceFiles[srcFileIndex].Metadata
		}
		if len(policyGenTemp.Spec.SourceFiles[srcFileIndex].Spec) != 0 {
			hasPatch = true
			newPatch["spec"] = policyGenTemp.Spec.SourceFiles[srcFileIndex].Spec
		}
		if len(policyGenTemp.Spec.SourceFiles[srcFileIndex].Status) != 0 {
			hasPatch = true
			newPatch["status"] = policyGenTemp.Spec.SourceFiles[srcFileIndex].Status
		}
		if hasPatch {
			newManifest.Patches = append(newManifest.Patches, newPatch)
		}

		pathRelativeToOutputDir := filepath.Join(outputDir, newManifest.Path)

		var ok bool
		annotations, err := fileutils.GetAnnotationsOnly(pathRelativeToOutputDir)
		if err != nil {
			fmt.Printf("could not get annotations from manifest:%s, err: %s\n", policyGenTemp.Spec.SourceFiles[srcFileIndex].FileName, err)
		}
		if wave, ok = annotations.Metadata.Annotations[waveAnnotationKey]; err == nil && ok &&
			wave != "" &&
			stringhelper.IsNumber(wave) {
			newPolicy.PolicyAnnotations[waveAnnotationKey] = wave
		}
		newPolicy.Manifests = append(newPolicy.Manifests, newManifest)
	}
	return newPolicy
}

// Maps miscellaneous PGT fields to the ACM Gen fields
func convertSimpleMiscellaneousFields(policyGenTemp *pgtformat.PolicyGenTemplate, acmGenTempConversion *acmformat.AcmGenTemplate, rootName string) {
	acmGenTempConversion.PolicyDefaults.Namespace = policyGenTemp.Metadata.Namespace
	acmGenTempConversion.PolicyDefaults.RemediationAction = "inform"
	acmGenTempConversion.Kind = "PolicyGenerator"
	acmGenTempConversion.APIVersion = "policy.open-cluster-management.io/v1"

	acmGenTempConversion.PolicyDefaults.Severity = "low"
	acmGenTempConversion.PolicyDefaults.NamespaceSelector = acmformat.NamespaceSelector{Exclude: []string{"kube-*"}, Include: []string{"*"}}
	acmGenTempConversion.PolicyDefaults.EvaluationInterval.Compliant = policyGenTemp.Spec.EvaluationInterval.Compliant
	acmGenTempConversion.PolicyDefaults.EvaluationInterval.NonCompliant = policyGenTemp.Spec.EvaluationInterval.NonCompliant

	acmGenTempConversion.Metadata.Name = rootName
	acmGenTempConversion.PlacementBindingDefaults.Name = rootName + "-placement-binding"
}
