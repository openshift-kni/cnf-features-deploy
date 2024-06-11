package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"flag"

	"github.com/openshift-kni/cnf-features-deploy/ztp/tools/pgt2acmpg/packages/acmformat"
	"github.com/openshift-kni/cnf-features-deploy/ztp/tools/pgt2acmpg/packages/fileutils"
	"github.com/openshift-kni/cnf-features-deploy/ztp/tools/pgt2acmpg/packages/labels"
	"github.com/openshift-kni/cnf-features-deploy/ztp/tools/pgt2acmpg/packages/patches"
	"github.com/openshift-kni/cnf-features-deploy/ztp/tools/pgt2acmpg/packages/pgtformat"
	"github.com/openshift-kni/cnf-features-deploy/ztp/tools/pgt2acmpg/packages/placement"
	"github.com/openshift-kni/cnf-features-deploy/ztp/tools/pgt2acmpg/packages/renderpolicies"
	"github.com/openshift-kni/cnf-features-deploy/ztp/tools/pgt2acmpg/packages/stringhelper"
	"gopkg.in/yaml.v3"
)

// processFlags Gets Kind and source CRs lists from user flags
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

//nolint:funlen
func main() {
	// Defines the input PGT directory or file
	var inputFile = flag.String("i", "", "the PGT input file")
	// Defines the output directory for generated ACM templates
	var outputDir = flag.String("o", "", "the ACMPG output Directory")
	// Defines the input schema file. Schema allows patching CRDs containing lists of objects
	var schema = flag.String("s", "", "the optional schema for all non base CRDs")
	// Defines list of manifest kinds to which to pre-render patches to
	var preRenderPatchKindString = flag.String("k", "", "the optional list of manifest kinds for which to pre-render patches")
	// Optionally generates ACM policies for PGT and ACMPG templates
	var generateACMPolicies = flag.Bool("g", false, "optionally generates ACM policies for PGT and ACMPG templates")
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

	// Get source CRs path and kind lists from user flags
	preRenderPatchKindList, preRenderSourceCRList := processFlags(inputFile, outputDir, preRenderPatchKindString, sourceCRs)

	// Copy Source CR to output PGT Dir
	var err error

	allFilesInInputPath, err := fileutils.GetAllYAMLFilesInPath(*inputFile)
	if err != nil {
		fmt.Printf("Could not get file list, err: %s", err)
		os.Exit(1)
	}
	fmt.Println("Got All Yaml in path")

	// Add a copy of source-crs in each folder with a kustomization.yaml
	if sourceCRs != nil && *sourceCRs != "" {
		err = fileutils.AddSourceCRsInTemplateDir(allFilesInInputPath, preRenderSourceCRList, *inputFile, *outputDir)
		if err != nil {
			fmt.Printf("Could not copy source-crs files in output dir, err: %s", err)
			os.Exit(1)
		}

		err = fileutils.AddSourceCRsInTemplateDir(allFilesInInputPath, preRenderSourceCRList, *inputFile, *inputFile)
		if err != nil {
			fmt.Printf("Could not copy source-crs files in input dir, err: %s", err)
			os.Exit(1)
		}
	}
	fmt.Println("Added source-crs for all kustomization files")

	// convert all PGT files
	policiesNamespaces := make(map[string]bool)
	err = convertAllPGTFiles(preRenderPatchKindList, allFilesInInputPath, inputFile, outputDir, schema, workaroundPlacement, policiesNamespaces)
	if err != nil {
		fmt.Printf("Could not convert PGT files, err: %s", err)
		os.Exit(1)
	}

	fmt.Printf("Converted all PGT files, found namespaces: %v\n", policiesNamespaces)

	if NSYAML != nil && *NSYAML != "" {
		err = fileutils.CopyAndProcessNSAndKustomizationYAML(*NSYAML, *inputFile, *outputDir, *skipDefaultPlacementBindings, policiesNamespaces)
		if err != nil {
			fmt.Printf("Could not post-process %s and %s files, err: %s", *NSYAML, fileutils.KustomizationFileName, err)
		}
	}
	fmt.Println("Copied Kustomization.yaml and updated NS file ")

	if *generateACMPolicies {
		err = renderpolicies.RenderAndWriteTemplateToYAML(*outputDir, renderpolicies.AcmPGRenderedYAMLFileName)
		if err != nil {
			fmt.Printf("Could not generate ACMPG policies, err: %s", err)
			os.Exit(1)
		}
		fmt.Println("Generated ACM policies from ACM policy generator")

		err = renderpolicies.RenderAndWriteTemplateToYAML(*inputFile, renderpolicies.PgtRenderedYAMLFileName)
		if err != nil {
			fmt.Printf("Could generate PGT policies, err: %s", err)
			os.Exit(1)
		}
		fmt.Println("Generated ACM policies from PGT")
	}
}

// convertAllPGTFiles loops through all PGT files in input directory and converts them to ACM policy generator format
func convertAllPGTFiles(preRenderPatchKindList, allFilesInInputPath []string, inputFile, outputDir, schema *string, workaroundPlacement *bool, policiesNamespaces map[string]bool) (err error) {
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
		err = convertPGTtoACM(*outputDir, *inputFile, file, filepath.Join(*outputDir, fileutils.PrefixLastPathComponent(relativePath, fileutils.ACMPrefix)), *schema, preRenderPatchKindList, workaroundPlacement, policiesNamespaces)
		if err != nil {
			return fmt.Errorf("failed to convert PGT to ACMPG, err=%s", err)
		}
	}
	return nil
}

// convertPGTtoACM Converts an PGT file to a ACMPG Template file
//
//nolint:funlen
func convertPGTtoACM(outputDir, baseDir, inputFile, outputFile, schema string, preRenderPatchKindList []string, workaroundPlacement *bool, policiesNamespaces map[string]bool) (err error) {
	policyGenFileContent, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("unable to open file: %s, err: %s ", inputFile, err)
	}
	policyGenTemp := pgtformat.PolicyGenTemplate{}

	err = yaml.Unmarshal(policyGenFileContent, &policyGenTemp)
	if err != nil {
		return fmt.Errorf("could not unmarshal PolicyGenTemplate data from %s: %s", inputFile, err)
	}
	policiesNamespaces[policyGenTemp.Metadata.Namespace] = true
	rootName := policyGenTemp.Metadata.Name
	acmPGTempConversion := acmformat.ACMPG{}

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
		newPolicy, err := convertPGTPolicyToACMPGPolicy(&policyGenTemp, baseDir, inputFile, rootName, policyName, outputDir, "")
		if err != nil {
			return err
		}
		acmPGTempConversion.Policies = append(acmPGTempConversion.Policies, newPolicy)
		var labelSelector map[string]interface{}
		labelSelector, err = labels.OutputGeneric(labels.LabelToSelector(policyGenTemp.Spec.BindingRules,
			policyGenTemp.Spec.BindingExcludedRules))
		if err != nil {
			return err
		}

		// Convert miscellaneous fields
		convertSimpleMiscellaneousFields(&policyGenTemp, &acmPGTempConversion, rootName)

		if !*workaroundPlacement {
			acmPGTempConversion.PolicyDefaults.Placement.LabelSelector = labelSelector
		} else {
			var ACMTemplateDir string
			_, ACMTemplateDir, err = fileutils.GetTemplatePaths(baseDir, inputFile, outputDir)

			if err != nil {
				return fmt.Errorf("failed to get template paths, err: %s", err)
			}

			// starts creating child policies as soon as the managed cluster starts installing
			var placementFilepathRelative string
			placementFilepathRelative, err = placement.GeneratePlacementFile(newPolicy.Name, acmPGTempConversion.PolicyDefaults.Namespace, ACMTemplateDir, labelSelector)
			if err != nil {
				return fmt.Errorf("error when generating placement file, err: %s", err)
			}
			acmPGTempConversion.PolicyDefaults.Placement.PlacementPath = placementFilepathRelative
		}

		// Apply patches on ACMPG since it is not yet supported officially
		if len(acmPGTempConversion.Policies) > 0 {
			for policyIndex := range acmPGTempConversion.Policies {
				for manifestIndex := range acmPGTempConversion.Policies[policyIndex].Manifests {
					err = RenderPatchesInManifestForSpecifiedKindsAndMCP(&policyGenTemp, &acmPGTempConversion, policyIndex, manifestIndex, baseDir, inputFile, outputDir, schema, preRenderPatchKindList)
					if err != nil {
						return fmt.Errorf("could not render patches in manifest, err: %s", err)
					}
				}
			}
		}
	}
	return writeConvertedTemplateToFile(&policyGenTemp, &acmPGTempConversion, outputFile)
}

// writeConvertedTemplateToFile write the ACM Policy Generator object to a yaml file
func writeConvertedTemplateToFile(policyGenTemp *pgtformat.PolicyGenTemplate, acmPGTempConversion *acmformat.ACMPG, outputFile string) (err error) {
	convertedContent, err := yaml.Marshal(acmPGTempConversion)
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

// RenderPatchesInManifestForSpecifiedKindsAndMCP Renders patches in manifest for kinds specified in flags
func RenderPatchesInManifestForSpecifiedKindsAndMCP(policyGenTemp *pgtformat.PolicyGenTemplate,
	acmPGTempConversion *acmformat.ACMPG,
	policyIndex, manifestIndex int, baseDir,
	pgtFilePath, outputDir, schema string,
	kindsToRender []string) (err error) {
	var relativePathTemplate, ACMTemplateDir string
	relativePathTemplate, ACMTemplateDir, err = fileutils.GetTemplatePaths(baseDir, pgtFilePath, outputDir)

	if err != nil {
		return fmt.Errorf("failed to get template paths, err: %s", err)
	}
	pathRelativeToOutputDir := filepath.Join(ACMTemplateDir, acmPGTempConversion.Policies[policyIndex].Manifests[manifestIndex].Path)
	renamedpathRelativeToOutputDir, err := fileutils.RenderMCPLines(pathRelativeToOutputDir, policyGenTemp.Spec.Mcp)
	if err != nil {
		return fmt.Errorf("cannot render MCP lines, err: %s", err)
	}
	relativeManifestPath, err := filepath.Rel(filepath.Join(outputDir, relativePathTemplate), renamedpathRelativeToOutputDir)
	if err != nil {
		return fmt.Errorf("cannot get the relative path from path: %s and directory: %s, err: %s", renamedpathRelativeToOutputDir, outputDir, err)
	}

	// we switch to using the renamed manifest file with MCP line commented out
	acmPGTempConversion.Policies[policyIndex].Manifests[manifestIndex].Path = relativeManifestPath
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
	if len(acmPGTempConversion.Policies[policyIndex].Manifests[manifestIndex].Patches) == 0 || schema == "" {
		return nil
	}

	patcher := patches.ManifestPatcher{Manifests: manifestFile, Patches: acmPGTempConversion.Policies[policyIndex].Manifests[manifestIndex].Patches}
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

	acmPGTempConversion.Policies[policyIndex].Manifests[manifestIndex].Patches = patchedFiles
	return nil
}

const (
	waveAnnotationKey = "ran.openshift.io/ztp-deploy-wave"
	sourceCrPrefix    = "source-crs"
)

// convertPGTPolicyToACMPGPolicy Converts PGT policy to ACMPG policy
//
//nolint:funlen
func convertPGTPolicyToACMPGPolicy(policyGenTemp *pgtformat.PolicyGenTemplate, baseDir, pgtPath, rootName, policyName, outputDir, forceWave string) (newPolicy acmformat.PolicyConfig, err error) {
	newPolicy.Name = rootName + "-" + policyName
	newPolicy.PolicyAnnotations = make(map[string]string)
	wave := ""
	for srcFileIndex := range policyGenTemp.Spec.SourceFiles {
		if policyGenTemp.Spec.SourceFiles[srcFileIndex].PolicyName != policyName {
			continue
		}
		if policyGenTemp.Spec.SourceFiles[srcFileIndex].FileName == "" {
			return newPolicy, fmt.Errorf("malformed PGT, could not parse manifest filename in PGT ns: %s name: %s",
				policyGenTemp.Metadata.Namespace, policyGenTemp.Metadata.Name)
		}

		newManifestPath := policyGenTemp.Spec.SourceFiles[srcFileIndex].FileName
		if !strings.HasPrefix(newManifestPath, sourceCrPrefix) {
			newManifestPath = filepath.Join(sourceCrPrefix, newManifestPath)
		}
		newManifest := acmformat.Manifest{Path: newManifestPath}

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

		newPolicy.PolicyAnnotations[waveAnnotationKey] = forceWave
		if forceWave == "" {
			relativePath, err := filepath.Rel(baseDir, filepath.Dir(pgtPath))
			if err != nil {
				return newPolicy, fmt.Errorf("failed to get relative path, err: %s", err)
			}
			pathRelativeToOutputDir := filepath.Join(outputDir, relativePath, newManifest.Path)
			var ok bool
			annotations, err := fileutils.GetAnnotationsOnly(pathRelativeToOutputDir)
			if err != nil {
				return newPolicy, fmt.Errorf("could not get annotations from manifest:%s in PGT ns: %s name: %s, err: %s", policyGenTemp.Spec.SourceFiles[srcFileIndex].FileName,
					policyGenTemp.Metadata.Namespace, policyGenTemp.Metadata.Name, err)
			}

			if wave, ok = annotations.Metadata.Annotations[waveAnnotationKey]; err == nil && ok &&
				wave != "" &&
				stringhelper.IsNumber(wave) {
				newPolicy.PolicyAnnotations[waveAnnotationKey] = wave
			}
		}

		newPolicy.Manifests = append(newPolicy.Manifests, newManifest)
	}
	return newPolicy, nil
}

// convertSimpleMiscellaneousFields Maps miscellaneous PGT fields to the ACMPG fields
func convertSimpleMiscellaneousFields(policyGenTemp *pgtformat.PolicyGenTemplate, acmPGTempConversion *acmformat.ACMPG, rootName string) {
	acmPGTempConversion.PolicyDefaults.Namespace = policyGenTemp.Metadata.Namespace
	acmPGTempConversion.PolicyDefaults.RemediationAction = pgtformat.Inform
	acmPGTempConversion.Kind = "PolicyGenerator"
	acmPGTempConversion.APIVersion = "policy.open-cluster-management.io/v1"

	acmPGTempConversion.PolicyDefaults.Severity = "low"
	acmPGTempConversion.PolicyDefaults.NamespaceSelector = acmformat.NamespaceSelector{Exclude: []string{"kube-*"}, Include: []string{"*"}}
	acmPGTempConversion.PolicyDefaults.EvaluationInterval.Compliant = policyGenTemp.Spec.EvaluationInterval.Compliant
	acmPGTempConversion.PolicyDefaults.EvaluationInterval.NonCompliant = policyGenTemp.Spec.EvaluationInterval.NonCompliant

	acmPGTempConversion.Metadata.Name = rootName
	acmPGTempConversion.PlacementBindingDefaults.Name = rootName + "-placement-binding"
}
