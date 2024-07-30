package fileutils

import (
	"fmt"
	"io"

	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	DefaultFileWritePermissions = 0o600
	DefaultDirWritePermissions  = 0o755
	ACMPrefix                   = "acm-"
	mcpPattern                  = "$mcp"
	SourceCRsDir                = "source-crs"
	KustomizationFileName       = "kustomization.yaml"
	NamespaceFileName           = "ns.yaml"
	defaultPlacementBindings    = `
---
apiVersion: cluster.open-cluster-management.io/v1beta2
kind: ManagedClusterSetBinding
metadata:
  name: global
  namespace: %s
spec:
  clusterSet: global
`
)

// CommentOutMCPLines Comments out lines containing the "$mcp" keyword
func CommentOutLinesWithPlaceholders(inputFile string) error {
	contents, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("unable to open file: %s, err: %s ", inputFile, err)
	}

	pattern := regexp.MustCompile(`.*: \$\S*`)

	// Split the file contents into lines
	lines := strings.Split(string(contents), "\n")

	// comment out every lines matching pattern
	var modifiedLines []string
	for _, line := range lines {
		if pattern.MatchString(line) {
			// Comment out the line by adding "//" at the beginning
			line = "# " + line
		}

		// Add the processed line to the result
		modifiedLines = append(modifiedLines, line)
	}

	// Join the modified lines
	modifiedString := strings.Join(modifiedLines, "\n")
	err = os.WriteFile(inputFile, []byte(modifiedString), DefaultFileWritePermissions)
	if err != nil {
		return fmt.Errorf("error writing to file: %s, err: %s", inputFile, err)
	}
	return nil
}

// RenderMCPLines Replaces the "$mcp" keyword with the mcp string (worker or master)
func RenderMCPLines(inputFile, mcp string) (outputFile string, err error) {
	const (
		mcpPattern = "$mcp"
	)

	contents, err := os.ReadFile(inputFile)
	if err != nil {
		return outputFile, fmt.Errorf("unable to open file: %s, err: %s ", inputFile, err)
	}

	outputFile = inputFile
	if strings.Contains(string(contents), mcpPattern) {
		contents = []byte(strings.ReplaceAll(string(contents), mcpPattern, mcp))
		outputFile = strings.TrimSuffix(inputFile, ".yaml") + "-MCP-" + mcp + ".yaml"
	}

	err = os.WriteFile(outputFile, contents, DefaultFileWritePermissions)
	if err != nil {
		return "", fmt.Errorf("error writing to file: %s, err: %s", outputFile, err)
	}

	err = CommentOutLinesWithPlaceholders(outputFile)
	if err != nil {
		return "", fmt.Errorf("error removing placeholders $** in file: %s, err: %s", outputFile, err)
	}

	fmt.Printf("Wrote converted ACM template: %s\n", outputFile)
	return outputFile, nil
}

// GetAllYAMLFilesInPath Gets All Yaml files in a path
func GetAllYAMLFilesInPath(path string) (files []string, err error) {
	err = filepath.Walk(path, func(filePath string, info os.FileInfo, _ error) error {
		if !info.IsDir() && (strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml")) {
			files = append(files, filePath)
		}
		return nil
	})

	return files, err
}

// PrefixLastPathComponent Prefixes the file referred to by a path with a given prefix
func PrefixLastPathComponent(originalPath, prefix string) string {
	dir, file := filepath.Split(originalPath)
	if dir == "" {
		// If the originalPath has no directory component, simply prefix the file name
		return prefix + file
	}
	return filepath.Join(dir, prefix+file)
}

// Type used to parse the Kind of a K8s Manifest
type KindType struct {
	Kind string `yaml:"kind"`
}

// GetManifestKind Gets the manifest kind from the file
func GetManifestKind(filePath string) (kindType KindType, err error) {
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		return kindType, fmt.Errorf("could not read %s: %s", filePath, err)
	}
	err = yaml.Unmarshal(yamlFile, &kindType)
	if err != nil {
		return kindType, fmt.Errorf("could not parse %s as yaml: %s", filePath, err)
	}
	return kindType, nil
}

type AnnotationsOnly struct {
	Metadata MetaData `yaml:"metadata"`
}

type MetaData struct {
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// GetAnnotationsOnly Gets the manifest kind from the file
func GetAnnotationsOnly(filePath string) (annotations AnnotationsOnly, err error) {
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		return annotations, fmt.Errorf("could not read %s: %s", filePath, err)
	}
	err = yaml.Unmarshal(yamlFile, &annotations)
	if err != nil {
		return annotations, fmt.Errorf("could not parse %s as yaml: %s", filePath, err)
	}
	return annotations, nil
}

// AddDefaultPlacementBindingsToNSFile Adds the default placement bindings for ACM Policy Generator
func AddDefaultPlacementBindingsToNSFile(namespaceFilePath, outputDir string, policiesNamespaces map[string]bool) (err error) {
	for namespace := range policiesNamespaces {
		fullNamespaceFilePath := filepath.Join(outputDir, namespaceFilePath)
		fileContent, err := os.ReadFile(fullNamespaceFilePath)
		if err != nil {
			return fmt.Errorf("could not read %s: %s", fullNamespaceFilePath, err)
		}

		fileContent = []byte(string(fileContent) + fmt.Sprintf(defaultPlacementBindings, namespace))
		err = os.WriteFile(fullNamespaceFilePath, fileContent, DefaultFileWritePermissions)
		if err != nil {
			return fmt.Errorf("error writing to file: %s, err: %s", fullNamespaceFilePath, err)
		}
		fmt.Printf("Added default placement binding for namespace: %s to: %s\n", namespace, fullNamespaceFilePath)
	}
	return nil
}

type Kustomization struct {
	Generators []string `yaml:"generators,omitempty"`
	Resources  []string `yaml:"resources,omitempty"`
	Bases      []string `yaml:"bases,omitempty"`
}

// RenameACMPGsInKustomization copy kustomization.yaml to output directory while renaming policies
func RenameACMPGsInKustomization(relativeFilePath, inputDir, outputDir string) (err error) {
	fileContent, err := os.ReadFile(filepath.Join(inputDir, relativeFilePath))
	if err != nil {
		return fmt.Errorf("could not read %s: %s", relativeFilePath, err)
	}

	// Unmarshal YAML data into a struct
	kustomization := Kustomization{}
	err = yaml.Unmarshal(fileContent, &kustomization)
	if err != nil {
		return fmt.Errorf("error unmarshalling yaml file: %s, err %v", relativeFilePath, err)
	}
	updatedKustomization := Kustomization{}
	for _, g := range kustomization.Generators {
		updatedKustomization.Generators = append(updatedKustomization.Generators, PrefixLastPathComponent(g, ACMPrefix))
	}
	updatedKustomization.Bases = append(updatedKustomization.Bases, kustomization.Bases...)
	// Copy all resources to destination directory
	for _, r := range kustomization.Resources {
		_, err = Copy(filepath.Join(inputDir, filepath.Dir(relativeFilePath), r), filepath.Join(outputDir, filepath.Dir(relativeFilePath), r))
		if err != nil {
			return fmt.Errorf("could not copy file from %s to %s, err:%s", filepath.Join(inputDir, r), filepath.Join(outputDir, r), err)
		}
		updatedKustomization.Resources = append(updatedKustomization.Resources, r)
		fmt.Printf("Wrote Kustomization resource: %s\n", filepath.Join(outputDir, r))
	}
	// Marshal the struct back to YAML
	outputContent, err := yaml.Marshal(updatedKustomization)
	if err != nil {
		return fmt.Errorf("error marshaling YAML content, err: %v", err)
	}
	outputKustomization := filepath.Join(outputDir, relativeFilePath)
	err = os.WriteFile(outputKustomization, outputContent, DefaultFileWritePermissions)
	if err != nil {
		return fmt.Errorf("error writing to file: %s, err: %s", outputKustomization, err)
	}
	fmt.Printf("Wrote Updated Kustomization file: %s\n", outputKustomization)
	return nil
}

// CopyAndProcessNSAndKustomizationYAML Performs post processing on ns.yaml and Kustomization.yaml files
func CopyAndProcessNSAndKustomizationYAML(nsFilePath, inputFile, outputDir string, skipUpdateNs bool, policiesNamespaces map[string]bool) (err error) {
	err = RenameACMPGsInAllKustomization(inputFile, outputDir)
	if err != nil {
		return fmt.Errorf("could not rename generators in kustomization file, err: %s", err)
	}
	// No need to update ns.yaml, exiting early
	if skipUpdateNs {
		return nil
	}
	err = AddDefaultPlacementBindingsToNSFile(nsFilePath, outputDir, policiesNamespaces)
	if err != nil {
		return fmt.Errorf("could not add placement bindings in NS file file, err: %s", err)
	}
	return nil
}

// Copy function found at https://opensource.com/article/18/6/copying-files-go
func Copy(src, dst string) (int64, error) {
	dstDir := filepath.Dir(dst)
	err := CreateIfNotExists(dstDir, DefaultDirWritePermissions)
	if err != nil {
		return 0, fmt.Errorf("could not create destination directory %s", dstDir)
	}

	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	if Exists(dst) {
		fmt.Printf("Skipping file: %s, already exists\n", dst)
		return 0, nil
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

// CopySourceCrs Copies source CRs to the output ACMPG Directory
func CopySourceCrs(outputDir string, preRenderSourceCRList []string) (err error) {
	for _, sourceCRsPath := range preRenderSourceCRList {
		err = CopyDirectory(sourceCRsPath, filepath.Join(outputDir, SourceCRsDir))
		if err != nil {
			fmt.Printf("Could not copy source-crs to %s directory, err: %s", filepath.Join(outputDir, SourceCRsDir), err)
			os.Exit(1)
		}
		fmt.Printf("Copied source-cr at %s to %s directory successfully\n", sourceCRsPath, filepath.Join(outputDir, SourceCRsDir))
	}
	return nil
}

// RenameACMPGsInAllKustomization Updates all the old PGT names to ACM PolicyGen format in kustomization file
func RenameACMPGsInAllKustomization(inputFile, outputDir string) (err error) {
	return filepath.Walk(inputFile, func(path string, info os.FileInfo, _ error) error {
		if !info.IsDir() && info.Name() == KustomizationFileName {
			relativePath, err := filepath.Rel(inputFile, path)
			if err != nil {
				return fmt.Errorf("failed processing relative path for kustomization.yaml at path: %s, err: %s", path, err)
			}
			err = RenameACMPGsInKustomization(relativePath, inputFile, outputDir)
			if err != nil {
				return fmt.Errorf("failed processing kustomization.yaml at path: %s, err: %s", path, err)
			}
		}
		return nil
	})
}

// GetTemplatePaths gets the path of
// - the template relative to the base ipnut directory
// - the output template full directory
func GetTemplatePaths(baseDir, pgtFilePath, outputDir string) (relativePathTemplate, acmTemplateDir string, err error) {
	relativePathTemplate, err = filepath.Rel(baseDir, filepath.Dir(pgtFilePath))
	if err != nil {
		return relativePathTemplate, acmTemplateDir, fmt.Errorf("failed to get relative path of template directory base: %s PGT file: %s, err: %s", baseDir, pgtFilePath, err)
	}
	acmTemplateDir = filepath.Join(outputDir, relativePathTemplate)
	return relativePathTemplate, acmTemplateDir, nil
}

// AddSourceCRsInTemplateDir Adds a source-crs directory in each directory containing a template (PGT or ACM policy generator)
func AddSourceCRsInTemplateDir(allFilesInInputPath, preRenderSourceCRList []string, inputFile, outputDir string) (err error) {
	pgtDirs := make(map[string]bool)
	for _, file := range allFilesInInputPath {
		var kindType KindType
		kindType, err = GetManifestKind(file)
		if err != nil {
			return fmt.Errorf("could not get manifest kind for file:%s, err: %s", file, err)
		}
		if kindType.Kind != "PolicyGenTemplate" {
			continue
		}
		pgtDirs[filepath.Dir(file)] = true
	}
	for directoryPath := range pgtDirs {
		relativePath, err := filepath.Rel(inputFile, directoryPath)
		if err != nil {
			return fmt.Errorf("could not get relative path for file: %s, base: %s, err: %s", directoryPath, inputFile, err)
		}
		err = CopySourceCrs(filepath.Join(outputDir, relativePath), preRenderSourceCRList)
		if err != nil {
			return fmt.Errorf("could not copy source-crs files, err: %s", err)
		}
	}
	return nil
}
