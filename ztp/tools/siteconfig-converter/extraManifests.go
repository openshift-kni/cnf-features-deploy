package main

import (
	"bytes"
	base64 "encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
	"unicode"

	"gopkg.in/yaml.v3"
)

// Constants for extra-manifests generation
const (
	localExtraManifestPath    = "extra-manifest"
	workloadPath              = "workload"
	workloadFile              = "03-workload-partitioning.yaml"
	workloadCrioFile          = "crio.conf"
	workloadKubeletFile       = "kubelet.conf"
	cpuset                    = "$cpuset"
	SNO                       = "sno"
	ZtpAnnotation             = "ran.openshift.io/ztp-gitops-generated"
	ZtpAnnotationDefaultValue = "{}"
)

// DirContainFiles represents a directory and its files
type DirContainFiles struct {
	Directory string
	Files     []fs.FileInfo
}

// resolveFilePath resolves a file path, checking if it's absolute first
func resolveFilePath(filePath string, basedir string) string {
	// If path is already absolute, return it as-is
	if filepath.IsAbs(filePath) {
		return filePath
	}
	// Check if the path exists as-is (could be relative to current working directory)
	if _, err := os.Stat(filePath); err == nil {
		return filePath
	}
	// Otherwise, resolve relative to basedir
	return filepath.Join(basedir, filePath)
}

// GetFiles returns file info for a path (file or directory)
func GetFiles(path string) ([]fs.FileInfo, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fileInfo.IsDir() {
		var files []fs.FileInfo
		results, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}

		// Translate []fs.DirEntry to []fs.FileInfo
		for _, result := range results {
			resultsInfo, err := result.Info()
			if err != nil {
				return nil, err
			}
			files = append(files, resultsInfo)
		}

		return files, nil
	}

	return []fs.FileInfo{fileInfo}, nil
}

// ReadFile reads a file and returns its contents
func ReadFile(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

// ReadExtraManifestResourceFile reads an extra manifest resource file
func ReadExtraManifestResourceFile(filePath string) ([]byte, error) {
	var dir = ""
	var err error = nil
	var ret []byte

	ex, err := os.Executable()
	if err != nil {
		return nil, err
	}
	dir = filepath.Dir(ex)

	ret, err = ReadFile(resolveFilePath(filePath, dir))

	// added fail safe for test runs as `os.Executable()` will fail for tests
	if err != nil {
		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}

		ret, err = ReadFile(resolveFilePath(filePath, dir))
	}
	return ret, err
}

// GetExtraManifestResourceDir gets the directory for extra manifest resources
func GetExtraManifestResourceDir(manifestsPath string) (string, error) {
	// If path is already absolute, return it as-is
	if filepath.IsAbs(manifestsPath) {
		return manifestsPath, nil
	}

	// Try to resolve relative to executable
	ex, err := os.Executable()
	if err != nil {
		// If we can't get executable path, try current working directory
		dir, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return resolveFilePath(manifestsPath, dir), nil
	}

	dir := filepath.Dir(ex)
	return resolveFilePath(manifestsPath, dir), nil
}

// addZTPAnnotationToManifest adds ZTP annotation to a manifest string
func addZTPAnnotationToManifest(manifestStr string) (string, error) {
	var data map[string]interface{}
	err := yaml.Unmarshal([]byte(manifestStr), &data)
	if err != nil {
		return manifestStr, fmt.Errorf("could not unmarshal string: %w", err)
	}

	// Add ZTP annotation
	if data["metadata"] == nil {
		data["metadata"] = make(map[string]interface{})
	}

	if data["metadata"].(map[string]interface{})["annotations"] == nil {
		data["metadata"].(map[string]interface{})["annotations"] = make(map[string]interface{})
	}

	data["metadata"].(map[string]interface{})["annotations"].(map[string]interface{})[ZtpAnnotation] = ZtpAnnotationDefaultValue

	out, err := yaml.Marshal(data)
	if err != nil {
		return manifestStr, fmt.Errorf("could not marshal data: %w", err)
	}
	return string(out), nil
}

// getWorkloadManifest generates workload manifest for SNO clusters
func getWorkloadManifest(fPath string, cpuSet string, role string) (string, interface{}, error) {
	filePath := filepath.Join(fPath, workloadPath)
	crio, err := ReadExtraManifestResourceFile(filepath.Join(filePath, workloadCrioFile))
	if err != nil {
		return "", nil, err
	}
	crioStr := string(crio)
	crioStr = strings.ReplaceAll(crioStr, cpuset, cpuSet)
	crioStr = base64.StdEncoding.EncodeToString([]byte(crioStr))

	kubelet, err := ReadExtraManifestResourceFile(filepath.Join(filePath, workloadKubeletFile))
	if err != nil {
		return "", nil, err
	}
	kubeletStr := string(kubelet)
	kubeletStr = strings.ReplaceAll(kubeletStr, cpuset, cpuSet)
	kubeletStr = base64.StdEncoding.EncodeToString([]byte(kubeletStr))

	workload, err := ReadExtraManifestResourceFile(filepath.Join(filePath, workloadFile))
	if err != nil {
		return "", nil, err
	}
	workloadStr := string(workload)
	workloadStr = strings.ReplaceAll(workloadStr, "$crio", crioStr)
	workloadStr = strings.ReplaceAll(workloadStr, "$k8s", kubeletStr)
	workloadStr = strings.ReplaceAll(workloadStr, "$mcp", role)

	workloadFileParts := append(strings.Split(workloadFile, "-"), "")
	copy(workloadFileParts[2:], workloadFileParts[1:])
	workloadFileParts[1] = role
	workloadFileForRole := strings.Join(workloadFileParts, "-")

	return workloadFileForRole, reflect.ValueOf(workloadStr).Interface(), nil
}

// getManifestFromTemplate processes a template file and returns the rendered manifest
func getManifestFromTemplate(templatePath, role string, data interface{}) (string, string, error) {
	baseName := filepath.Base(templatePath)
	renderedName := fmt.Sprintf("%s-%s", role, strings.TrimSuffix(baseName, ".tmpl"))
	tStr, err := ReadExtraManifestResourceFile(templatePath)
	if err != nil {
		return "", "", err
	}
	t, err := template.New(baseName).Parse(string(tStr))
	if err != nil {
		return "", "", err
	}
	var output bytes.Buffer
	err = t.Execute(&output, struct {
		Role string
		Data interface{}
	}{
		Role: role,
		Data: data,
	})
	if err != nil {
		return "", "", err
	}
	// Ensure there's non-whitespace content
	for _, r := range output.String() {
		if !unicode.IsSpace(r) {
			return renderedName, output.String(), nil
		}
	}
	// Output is all whitespace; return nil instead
	return "", "", nil
}

// getExtraManifestMaps returns 2 maps and error
// first map consists: key as fileName, value as file content
// second map contains: key as machineConfigs filename, value as true/false
func getExtraManifestMaps(roles map[string]bool, clusterSpec Cluster, filePath ...string) (map[string]interface{}, map[string]bool, error) {
	var (
		files []fs.FileInfo
		err   error
	)

	dirFiles := &DirContainFiles{}
	dirFilesArray := []DirContainFiles{}
	dataMap := make(map[string]interface{})
	// Note: doNotMerge is not used as MergeManifests is not implemented
	doNotMerge := make(map[string]bool)

	for _, p := range filePath {
		files, err = GetFiles(p)
		if err != nil {
			return nil, nil, err
		}
		dirFiles = &DirContainFiles{
			Directory: p,
			Files:     files,
		}
		dirFilesArray = append(dirFilesArray, *dirFiles)
	}

	for _, v := range dirFilesArray {
		for _, file := range v.Files {
			if file.IsDir() || file.Name()[0] == '.' {
				continue
			}

			filePath := filepath.Join(v.Directory, file.Name())

			if strings.HasSuffix(file.Name(), ".tmpl") {
				// For templates, we can inject the roles directly
				// Assumes that templates that don't care about roles take precautions that they will be called per role.
				for role := range roles {
					filename, value, err := getManifestFromTemplate(filePath, role, clusterSpec)
					if err != nil {
						return dataMap, doNotMerge, err
					}
					if value != "" {
						value, err = addZTPAnnotationToManifest(value)
						if err != nil {
							return dataMap, doNotMerge, err
						}
						dataMap[filename] = value
						// Note: doNotMerge tracking removed as MergeManifests is not implemented
					}
				}
			} else {
				// This is a pure passthrough, assuming any static files for both 'master' and 'worker' have their contents set up properly.
				manifestFile, err := ReadExtraManifestResourceFile(filePath)
				if err != nil {
					return dataMap, doNotMerge, err
				}
				if len(manifestFile) != 0 {
					manifestFileStr, err := addZTPAnnotationToManifest(string(manifestFile))
					if err != nil {
						return dataMap, doNotMerge, err
					}
					dataMap[file.Name()] = manifestFileStr
				}
			}
		}
	}

	// Adding workload partitions MC only for SNO clusters.
	if len(clusterSpec.Nodes) == 1 {
		for node := range clusterSpec.Nodes {
			cpuSet := clusterSpec.Nodes[node].Cpuset
			role := clusterSpec.Nodes[node].Role
			if cpuSet != "" {
				for _, v := range dirFilesArray {
					for _, file := range v.Files {
						if file.Name() == workloadPath {
							k, v, err := getWorkloadManifest(v.Directory, cpuSet, role)
							if err != nil {
								errStr := fmt.Sprintf("Error could not read WorkloadManifest %s %s\n", clusterSpec.ClusterName, err)
								return dataMap, doNotMerge, errors.New(errStr)
							} else if v.(string) != "" {
								data, err := addZTPAnnotationToManifest(v.(string))
								if err != nil {
									return dataMap, doNotMerge, err
								}
								dataMap[k] = data
								// Note: doNotMerge tracking removed as MergeManifests is not implemented
							}
						}
					}
				}
			}
		}
	}

	return dataMap, doNotMerge, err
}

// filterExtraManifests filters manifests based on filter configuration
func filterExtraManifests(dataMap map[string]interface{}, filter *Filter) (map[string]interface{}, error) {
	// return if there's no filter initialized
	if filter == nil {
		return dataMap, nil
	}

	inclusionDefaultInclude := "include"
	inclusionDefaultExclude := "exclude"
	// use this internally for faster comparison
	var excludeAllByDefault bool

	// default value is include. treat use of `exclude` as an advanced feature
	if filter.InclusionDefault == nil || strings.EqualFold(*filter.InclusionDefault, inclusionDefaultInclude) {
		excludeAllByDefault = false
	} else if strings.EqualFold(*filter.InclusionDefault, inclusionDefaultExclude) {
		excludeAllByDefault = true
	} else {
		errStr := fmt.Sprintf("acceptable values for inclusionDefault are %s and %s. You have entered %s", inclusionDefaultInclude, inclusionDefaultExclude, *filter.InclusionDefault)
		return dataMap, errors.New(errStr)
	}

	// helper to create the debug msg
	getDataMapFileNameInStrings := func(dataMap map[string]interface{}) string {
		var files []string
		for s := range dataMap {
			files = append(files, s)
		}
		stringFiles := strings.Join(files, ",")
		return stringFiles
	}

	if excludeAllByDefault {
		// in `exclude` mode

		// check if include list is empty
		if len(filter.Exclude) > 0 {
			return dataMap, errors.New("when InclusionDefault is set to exclude, exclude list can not have entries")
		}

		temp := make(map[string]interface{})
		for _, fileToInclude := range filter.Include {
			value, exists := dataMap[fileToInclude]
			if exists {
				temp[fileToInclude] = value
			} else {
				errStr := fmt.Sprintf("Filename %s under include array is invalid. Valid files names are: %s",
					fileToInclude, getDataMapFileNameInStrings(dataMap))
				return dataMap, errors.New(errStr)
			}
		}
		return temp, nil
	} else {
		// in `include` mode

		// check if exclude list is empty
		if len(filter.Include) > 0 {
			return dataMap, errors.New("when InclusionDefault is set to include, include list can not have entries")
		}

		// remove the files using exclude list
		for _, fileToExclude := range filter.Exclude {
			_, exists := dataMap[fileToExclude]
			if exists {
				delete(dataMap, fileToExclude)
			} else {
				errStr := fmt.Sprintf("Filename %s under exclude array is invalid. Valid files names are: %s", fileToExclude, getDataMapFileNameInStrings(dataMap))
				return dataMap, errors.New(errStr)
			}
		}
	}

	return dataMap, nil
}

// getExtraManifest generates extra manifests for a cluster
func getExtraManifest(dataMap map[string]interface{}, clusterSpec Cluster, inputFileDir string) (map[string]interface{}, error) {
	// Figure out the list of node roles we need to support in this cluster
	roles := map[string]bool{}
	var err error

	for _, node := range clusterSpec.Nodes {
		roles[node.Role] = true
	}

	if clusterSpec.ExtraManifests.SearchPaths != nil {
		// Resolve relative paths in searchPaths relative to the input file directory
		resolvedPaths := make([]string, 0, len(*clusterSpec.ExtraManifests.SearchPaths))
		for _, path := range *clusterSpec.ExtraManifests.SearchPaths {
			resolvedPath := path
			if !filepath.IsAbs(path) {
				// Resolve relative to input file directory
				resolvedPath = filepath.Join(inputFileDir, path)
				// Clean the path to remove any ".." or "." components
				resolvedPath = filepath.Clean(resolvedPath)
			}
			resolvedPaths = append(resolvedPaths, resolvedPath)
		}
		dataMap, _, err = getExtraManifestMaps(roles, clusterSpec, resolvedPaths...)
		if err != nil {
			return dataMap, err
		}
	} else {
		containerPath, err := GetExtraManifestResourceDir(localExtraManifestPath)
		if err != nil {
			return dataMap, err
		}
		// Check if the directory exists before trying to use it
		if _, err := os.Stat(containerPath); err == nil {
			dataMap, _, err = getExtraManifestMaps(roles, clusterSpec, containerPath)
			if err != nil {
				return dataMap, err
			}
		}
		// If directory doesn't exist, continue without error (no default extra-manifests)
	}

	// Adding End User Extra-manifest
	if clusterSpec.ExtraManifests.SearchPaths == nil && clusterSpec.ExtraManifestPath != "" {
		// Resolve extraManifestPath relative to input file directory if it's relative
		extraManifestPath := clusterSpec.ExtraManifestPath
		if !filepath.IsAbs(extraManifestPath) {
			extraManifestPath = filepath.Join(inputFileDir, extraManifestPath)
			extraManifestPath = filepath.Clean(extraManifestPath)
		}
		files, err := GetFiles(extraManifestPath)
		if err != nil {
			return dataMap, fmt.Errorf("failed to access extraManifestPath %s (resolved from %s): %w", extraManifestPath, clusterSpec.ExtraManifestPath, err)
		}
		for _, file := range files {
			if file.IsDir() || file.Name()[0] == '.' {
				continue
			}

			// return and fail if one of the end user extra-manifest has same name as the pre-defined extra-manifest.
			if dataMap[file.Name()] != nil {
				errStr := fmt.Sprintf("Pre-defined extra-manifest cannot be over written %s", file.Name())
				return dataMap, errors.New(errStr)
			}

			filePath := filepath.Join(extraManifestPath, file.Name())
			manifestFile, err := ReadFile(filePath)
			if err != nil {
				return dataMap, err
			}

			if len(manifestFile) != 0 {
				manifestFileStr, err := addZTPAnnotationToManifest(string(manifestFile))
				if err != nil {
					return dataMap, err
				}
				dataMap[file.Name()] = manifestFileStr
			}
		}
	}

	//filter CRs
	dataMap, err = filterExtraManifests(dataMap, clusterSpec.ExtraManifests.Filter)
	if err != nil {
		return dataMap, fmt.Errorf("could not filter %s.%s: %w", clusterSpec.ClusterName, clusterSpec.ExtraManifestPath, err)
	}

	// Note: MergeManifests is not included here as it requires machine-config-operator dependencies
	// If mergeDefaultMachineConfigs is needed, it should be handled separately

	return dataMap, nil
}

// generateExtraManifests generates extra manifests for a SiteConfig and writes them to the output directory
func generateExtraManifests(siteConfig *SiteConfig, outputDir string, inputFileDir string) error {
	if len(siteConfig.Spec.Clusters) == 0 {
		return fmt.Errorf("no clusters found in SiteConfig")
	}

	// Process each cluster
	for _, cluster := range siteConfig.Spec.Clusters {
		// Generate extra manifests for this cluster
		dataMap := make(map[string]interface{})
		extraManifests, err := getExtraManifest(dataMap, cluster, inputFileDir)
		if err != nil {
			return fmt.Errorf("failed to generate extra manifests for cluster %s: %w", cluster.ClusterName, err)
		}

		// Write each manifest to a file
		for filename, content := range extraManifests {
			manifestContent, ok := content.(string)
			if !ok {
				return fmt.Errorf("invalid manifest content type for %s", filename)
			}

			outputFile := filepath.Join(outputDir, filename)
			if err := os.WriteFile(outputFile, []byte(manifestContent), 0644); err != nil {
				return fmt.Errorf("failed to write manifest file %s: %w", outputFile, err)
			}
		}
	}

	return nil
}
