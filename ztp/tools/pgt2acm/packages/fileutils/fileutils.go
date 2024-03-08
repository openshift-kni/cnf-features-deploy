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
)

// Comments out lines containing the "$mcp" keyword
func CommentOutMCPLines(inputFile string) (outputFile string, patchList []map[string]interface{}, err error) {
	contents, err := os.ReadFile(inputFile)
	if err != nil {
		return outputFile, patchList, fmt.Errorf("unable to open file: %s, err: %s ", inputFile, err)
	}

	pattern := regexp.MustCompile(fmt.Sprintf(`.*%s.*`, regexp.QuoteMeta(mcpPattern)))

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
	outputFile = strings.TrimSuffix(inputFile, ".yaml") + "-SetSelector.yaml"
	err = os.WriteFile(outputFile, []byte(modifiedString), DefaultFileWritePermissions)
	if err != nil {
		return "", patchList, fmt.Errorf("error writing to file: %s, err: %s", inputFile, err)
	}
	fmt.Printf("Wrote converted ACM template: %s\n", outputFile)
	return outputFile, patchList, nil
}

// Replaces the "$mcp" keyword with the mcp string (worker or master)
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
		return "", fmt.Errorf("error writing to file: %s, err: %s", inputFile, err)
	}
	fmt.Printf("Wrote converted ACM template: %s\n", outputFile)
	return outputFile, nil
}

// Gets All Yaml files in a path
func GetAllYAMLFilesInPath(path string) (files []string, err error) {
	err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		//		if !info.IsDir() {
		if !info.IsDir() && (strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml")) {
			files = append(files, filePath)
		}
		return nil
	})

	return files, err
}

// Prefixes the file referred to by a path with a given prefix
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

// Gets the manifest kind from the file
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

// Gets the manifest kind from the file
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

const defaultPlacementBindings = `
---
apiVersion: cluster.open-cluster-management.io/v1beta2
kind: ManagedClusterSetBinding
metadata:
  name: global
  namespace: ztp-common
spec:
  clusterSet: global
---
apiVersion: cluster.open-cluster-management.io/v1beta2
kind: ManagedClusterSetBinding
metadata:
  name: global
  namespace: ztp-group
spec:
  clusterSet: global
---
apiVersion: cluster.open-cluster-management.io/v1beta2
kind: ManagedClusterSetBinding
metadata:
  name: global
  namespace: ztp-site
spec:
  clusterSet: global
`

func AddDefaultPlacementBindingsToNSFile(namespaceFilePath, outputDir string) (err error) {
	fullNamespaceFilePath := filepath.Join(outputDir, namespaceFilePath)
	fileContent, err := os.ReadFile(fullNamespaceFilePath)
	if err != nil {
		return fmt.Errorf("could not read %s: %s", fullNamespaceFilePath, err)
	}

	fileContent = []byte(string(fileContent) + defaultPlacementBindings)
	err = os.WriteFile(fullNamespaceFilePath, fileContent, DefaultFileWritePermissions)
	if err != nil {
		return fmt.Errorf("error writing to file: %s, err: %s", fullNamespaceFilePath, err)
	}
	fmt.Printf("Added default placement binding to:%s\n", fullNamespaceFilePath)
	return nil
}

type Kustomization struct {
	Generators []string `yaml:"generators"`
	Resources  []string `yaml:"resources"`
}

func RenameACMGenTemplatesInKustomization(inputFile, outputDir string) (err error) {
	inputKustomization := filepath.Join(inputFile, KustomizationFileName)
	fileContent, err := os.ReadFile(inputKustomization)
	if err != nil {
		return fmt.Errorf("could not read %s: %s", inputKustomization, err)
	}

	// Unmarshal YAML data into a struct
	kustomization := Kustomization{}
	err = yaml.Unmarshal(fileContent, &kustomization)
	if err != nil {
		return fmt.Errorf("error unmarshaling yaml file: %s, err %v", inputKustomization, err)
	}
	updatedKustomization := Kustomization{}
	for _, g := range kustomization.Generators {
		updatedKustomization.Generators = append(updatedKustomization.Generators, PrefixLastPathComponent(g, ACMPrefix))
	}
	// Copy all resources to destination directory
	for _, r := range kustomization.Resources {
		_, err = Copy(filepath.Join(inputFile, r), filepath.Join(outputDir, r))
		if err != nil {
			return fmt.Errorf("could not copy file from %s to %s", filepath.Join(inputFile, r), filepath.Join(outputDir, r))
		}
		updatedKustomization.Resources = append(updatedKustomization.Resources, r)
		fmt.Printf("Wrote Kustomization resource: %s\n", filepath.Join(outputDir, r))
	}
	// Marshal the struct back to YAML
	outputContent, err := yaml.Marshal(updatedKustomization)
	if err != nil {
		return fmt.Errorf("error marshaling YAML content, err: %v", err)
	}
	outputKustomization := filepath.Join(outputDir, KustomizationFileName)
	err = os.WriteFile(outputKustomization, outputContent, DefaultFileWritePermissions)
	if err != nil {
		return fmt.Errorf("error writing to file: %s, err: %s", outputKustomization, err)
	}
	fmt.Printf("Wrote Updated Kustomization file: %s\n", outputKustomization)
	return nil
}

func CopyAndProcessNSAndKustomizationYAML(nsFilePath, inputFile, outputDir string, skipUpdateNs bool) (err error) {
	err = RenameACMGenTemplatesInKustomization(inputFile, outputDir)
	if err != nil {
		return fmt.Errorf("could not rename generators in kustomization file, err: %s", err)
	}
	// No need to update ns.yaml, exiting early
	if skipUpdateNs {
		return nil
	}
	err = AddDefaultPlacementBindingsToNSFile(nsFilePath, outputDir)
	if err != nil {
		return fmt.Errorf("could not add placement bindings in NS file file, err: %s", err)
	}
	return nil
}

// function found at https://opensource.com/article/18/6/copying-files-go
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

func CopySourceCrs(inputFile, outputDir string, preRenderSourceCRList []string) (err error) {
	for _, sourceCRsPath := range preRenderSourceCRList {
		err = CopyDirectory(sourceCRsPath, filepath.Join(outputDir, SourceCRsDir))
		if err != nil {
			fmt.Printf("Could not copy source-crs to %s directory, err: %s", err, filepath.Join(outputDir, SourceCRsDir))
			os.Exit(1)
		}
		fmt.Printf("Copied source-cr at %s to %s directory successfully\n", sourceCRsPath, filepath.Join(outputDir, SourceCRsDir))

		err = CopyDirectory(sourceCRsPath, filepath.Join(inputFile, SourceCRsDir))
		if err != nil {
			fmt.Printf("Could not copy source-crs to %s directory, err: %s", err, filepath.Join(inputFile, SourceCRsDir))
			os.Exit(1)
		}
		fmt.Printf("Copied source-cr at %s to %s directory successfully\n", sourceCRsPath, filepath.Join(inputFile, SourceCRsDir))
	}
	return nil
}
