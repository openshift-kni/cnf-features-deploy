package siteConfig

import (
	"bytes"
	base64 "encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
	"unicode"

	yaml "gopkg.in/yaml.v3"
)

type SiteConfigBuilder struct {
	SourceClusterCRs           []interface{}
	scBuilderExtraManifestPath string
}

func NewSiteConfigBuilder() (*SiteConfigBuilder, error) {
	scBuilder := SiteConfigBuilder{scBuilderExtraManifestPath: localExtraManifestPath}

	return &scBuilder, nil
}

func (scbuilder *SiteConfigBuilder) loadSourceClusterCRs(clusterCRs string) error {
	clusterCRsYamls, err := scbuilder.splitYamls([]byte(clusterCRs))
	if err != nil {
		return err
	}
	scbuilder.SourceClusterCRs = make([]interface{}, len(clusterCRsYamls))
	for id, clusterCRsYaml := range clusterCRsYamls {
		var clusterCR interface{}
		err := yaml.Unmarshal(clusterCRsYaml, &clusterCR)

		if err != nil {
			return err
		}
		scbuilder.SourceClusterCRs[id] = clusterCR
	}

	return nil
}

func (scbuilder *SiteConfigBuilder) SetLocalExtraManifestPath(path string) {
	scbuilder.scBuilderExtraManifestPath = path
}

func (scbuilder *SiteConfigBuilder) Build(siteConfigTemp SiteConfig) (map[string][]interface{}, error) {

	switch apiVersion := siteConfigTemp.ApiVersion; apiVersion {
	case siteConfigAPIV1:
		err := scbuilder.loadSourceClusterCRs(clusterCRsV1)
		if err != nil {
			return nil, err
		}
	case siteConfigAPIV2:
		err := scbuilder.loadSourceClusterCRs(clusterCRsV2)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("Not valid siteConfig ApiVersion " + siteConfigTemp.ApiVersion)
	}

	clustersCRs := make(map[string][]interface{})

	err := scbuilder.validateSiteConfig(siteConfigTemp)
	if err != nil {
		return clustersCRs, err
	}

	for id, cluster := range siteConfigTemp.Spec.Clusters {
		clusterCRs, err := scbuilder.getClusterCRs(id, siteConfigTemp)
		if err != nil {
			return clustersCRs, err
		}

		clustersCRs[siteConfigTemp.Metadata.Name+"/"+cluster.ClusterName] = clusterCRs
	}

	return clustersCRs, nil
}

func (scbuilder *SiteConfigBuilder) validateSiteConfig(siteConfigTemp SiteConfig) error {
	clusters := make(map[string]bool)
	for id, cluster := range siteConfigTemp.Spec.Clusters {
		if cluster.ClusterName == "" {
			return errors.New("Error: Missing cluster name at site " + siteConfigTemp.Metadata.Name)
		}
		if clusters[siteConfigTemp.Metadata.Name+"/"+cluster.ClusterName] {
			return errors.New("Error: Repeated Cluster Name " + siteConfigTemp.Metadata.Name + "/" + cluster.ClusterName)
		}

		var err error
		siteConfigTemp.Spec.Clusters[id].NetworkType, err = agentClusterInstallAnnotation(cluster.NetworkType, cluster.InstallConfigOverrides)
		if err != nil {
			return errors.New("Error: " + err.Error())
		}

		if siteConfigTemp.Spec.ClusterImageSetNameRef == "" && siteConfigTemp.Spec.Clusters[id].ClusterImageSetNameRef == "" {
			return errors.New("Error: Site and cluster clusterImageSetNameRef cannot be empty " + siteConfigTemp.Metadata.Name + "/" + cluster.ClusterName)
		}
		// If cluster has not set a clusterImageSetNameRef we use the site one
		if siteConfigTemp.Spec.Clusters[id].ClusterImageSetNameRef == "" {
			siteConfigTemp.Spec.Clusters[id].ClusterImageSetNameRef = siteConfigTemp.Spec.ClusterImageSetNameRef
		}

		clusters[siteConfigTemp.Metadata.Name+"/"+cluster.ClusterName] = true
		for _, node := range cluster.Nodes {
			if err := node.IronicInspect.IsValid(); err != nil {
				return errors.New("Error: " + err.Error() + siteConfigTemp.Metadata.Name + "/" + cluster.ClusterName + "/" + node.HostName)
			}
		}
	}
	return nil
}

func (scbuilder *SiteConfigBuilder) getClusterCRs(clusterId int, siteConfigTemp SiteConfig) ([]interface{}, error) {
	var clusterCRs []interface{}
	validKinds := make(map[string]bool)
	validNodeKinds := make(map[string]bool)
	cluster := siteConfigTemp.Spec.Clusters[clusterId]

	// Get all extra manifest files
	extraManifestMap := make(map[string]interface{})
	extraManifestMap, err := scbuilder.getExtraManifest(extraManifestMap, cluster)
	if err != nil {
		// Will return and fail if the end user extra-manifest having issues.
		log.Printf("Error could not create extra-manifest %s.%s %s\n", cluster.ClusterName, cluster.ExtraManifestPath, err)
		return clusterCRs, err
	}

	// Generate extra manifest CRs only
	if cluster.ExtraManifestOnly {
		for _, manifest := range extraManifestMap {
			dataResult := make(map[string]interface{})
			err := yaml.Unmarshal([]byte(manifest.(string)), &dataResult)
			if err != nil {
				log.Printf("Error: could not unmarshal string:(%s) (%s)\n", manifest.(string), err)
				return clusterCRs, err
			}
			clusterCRs = append(clusterCRs, dataResult)
		}
		return clusterCRs, nil
	}

	for _, cr := range scbuilder.SourceClusterCRs {
		mapSourceCR := cr.(map[string]interface{})
		kind := mapSourceCR["kind"].(string)
		validKinds[kind] = true

		if kind == "BareMetalHost" || kind == "NMStateConfig" || kind == "HostFirmwareSettings" {
			// node-level CR (create one for each node)
			validNodeKinds[kind] = true
			for ndId, node := range cluster.Nodes {
				instantiatedCR, err := scbuilder.instantiateCR(fmt.Sprintf("node %s in cluster %s", node.HostName, cluster.ClusterName),
					mapSourceCR,
					func(kind string) (string, bool) {
						return node.CrTemplateSearch(kind, &cluster, &siteConfigTemp.Spec)
					},
					func(source map[string]interface{}) (map[string]interface{}, error) {
						return scbuilder.getClusterCR(clusterId, siteConfigTemp, source, ndId)
					},
				)
				if err != nil {
					return clusterCRs, err
				}

				// BZ 2028510 -- Empty NMStateConfig causes issues and
				// should simply be left out.
				if kind == "NMStateConfig" && node.nodeNetworkIsEmpty() {
					// noop, leave the empty NMStateConfig CR out of the generated set
				} else if kind == "HostFirmwareSettings" {
					if filePath := node.BiosFileSearch(&cluster, &siteConfigTemp.Spec); filePath != "" {
						err := populateSpec(filePath, instantiatedCR)
						if err != nil {
							return clusterCRs, err
						}
						clusterCRs = append(clusterCRs, instantiatedCR)
					}
				} else {
					if kind == "BareMetalHost" {
						// Ironic inspection is enabled by default, delete the inspect annotation if it is not disabled
						instantiatedCR = deleteInspectAnnotation(instantiatedCR)
						// Transform the node label annotation
						if len(node.NodeLabels) > 0 {
							instantiatedCR = transformNodeLabelAnnotation(instantiatedCR)
						}
					}
					clusterCRs = append(clusterCRs, instantiatedCR)
				}
			}
		} else {
			instantiatedCR, err := scbuilder.instantiateCR(fmt.Sprintf("cluster %s", cluster.ClusterName),
				mapSourceCR,
				func(kind string) (string, bool) {
					return cluster.CrTemplateSearch(kind, &siteConfigTemp.Spec)
				},
				func(source map[string]interface{}) (map[string]interface{}, error) {
					return scbuilder.getClusterCR(clusterId, siteConfigTemp, source, -1)
				},
			)
			if err != nil {
				return clusterCRs, err
			}

			// cluster-level CR
			if kind == "ConfigMap" {
				// For ConfigMap, add all the ExtraManifest files to it before doing further instantiation:
				data := instantiatedCR["data"]
				if data != nil {
					for k, v := range data.(map[string]interface{}) {
						extraManifestMap[k] = v
					}
				}
				instantiatedCR["data"] = extraManifestMap
			}
			clusterCRs = append(clusterCRs, instantiatedCR)
		}
	}

	// Double-check that the user didn't ask us to override any invalid object types:
	err = siteConfigTemp.areAllOverridesValid(&validKinds, &validNodeKinds)
	if err != nil {
		return clusterCRs, err
	}

	deprecationWarnings := getDeprecationWarnings(cluster)

	return addZTPAnnotationToCRs(clusterCRs, deprecationWarnings)
}

func (scbuilder *SiteConfigBuilder) instantiateCR(target string, originalTemplate map[string]interface{}, overrideSearch func(kind string) (string, bool), applyTemplate func(map[string]interface{}) (map[string]interface{}, error)) (map[string]interface{}, error) {
	// Instantiate the CR based on the original original CR
	originalCR, err := applyTemplate(originalTemplate)
	if err != nil {
		return map[string]interface{}{}, err
	}

	kind := originalCR["kind"].(string)
	overridePath, overrideFound := overrideSearch(kind)
	if !overrideFound {
		// No override provided; return the instantiation of the originalCR template
		return originalCR, nil
	}

	log.Printf("Overriding %s with %q for %s", kind, overridePath, target)
	override := make(map[string]interface{})
	overrideBytes, err := ReadFile(overridePath)
	if err != nil {
		log.Printf("Could not read %q: %v", overridePath, err)
		return override, err
	}
	err = yaml.Unmarshal(overrideBytes, &override)
	if err != nil {
		log.Printf("Could not parse %q: %v", overridePath, err)
		return override, err
	}
	if override["kind"] != kind {
		return override, fmt.Errorf("Override template kind %q in %q does not match expected kind %q", override["kind"], overridePath, kind)
	}
	overriddenCR, err := applyTemplate(override)
	if err != nil {
		return override, err
	}

	// Sanity-check the resulting metadata to ensure it's valid compared to the non-overridden original CR:
	originalMetadata := originalCR["metadata"].(map[string]interface{})

	// Sanity-check the overridden metadata object to ensure that it's instantiated correctly
	overriddenMetadata, ok := overriddenCR["metadata"].(map[string]interface{})
	if !ok {
		return override, fmt.Errorf("Overriden template metadata in %q is not specified!", overridePath)
	}

	for _, field := range []string{"name", "namespace"} {
		if originalMetadata[field] != overriddenMetadata[field] {
			return overriddenCR, fmt.Errorf("Overridden template metadata.%s %q does not match expected value %q", field, overriddenMetadata[field], originalMetadata[field])
		}
	}
	originalAnnotations := originalMetadata["annotations"].(map[string]interface{})

	// Sanity-check the overridden metadata annotations
	overriddenAnnotations, ok := overriddenMetadata["annotations"].(map[string]interface{})
	if !ok {
		return override, fmt.Errorf("Overriden template metadata annotations in %q is not specified!", overridePath)
	}

	// Validate the the argocd annotation
	argocdAnnotation := "argocd.argoproj.io/sync-wave"
	if originalAnnotations[argocdAnnotation] != overriddenAnnotations[argocdAnnotation] {
		return overriddenCR, fmt.Errorf("Overridden template metadata.annotations[%q] %q does not match expected value %q", argocdAnnotation, overriddenAnnotations[argocdAnnotation], originalAnnotations[argocdAnnotation])
	}
	return overriddenCR, nil
}

func (scbuilder *SiteConfigBuilder) getClusterCR(clusterId int, siteConfigTemp SiteConfig, mapSourceCR map[string]interface{}, nodeId int) (map[string]interface{}, error) {
	mapIntf := make(map[string]interface{})

	for k, v := range mapSourceCR {
		switch reflect.ValueOf(v).Kind() {
		case reflect.Map:
			value, err := scbuilder.getClusterCR(clusterId, siteConfigTemp, v.(map[string]interface{}), nodeId)
			if err != nil {
				return mapIntf, err
			}
			mapIntf[k] = value
		case reflect.Slice:
			sliceValues := make([]interface{}, 0)
			for _, item := range v.([]interface{}) {
				val, err := scbuilder.getClusterCR(clusterId, siteConfigTemp, item.(map[string]interface{}), nodeId)
				if err != nil {
					return mapIntf, err
				}
				sliceValues = append(sliceValues, val)
			}
			mapIntf[k] = sliceValues
		case reflect.String:
			if strings.HasPrefix(v.(string), "{{") && strings.HasSuffix(v.(string), "}}") {
				// We can be cleaner about this, but this translation is minimally invasive for 4.10:
				key, err := translateTemplateKey(v.(string))
				if err != nil {
					return nil, err
				}
				valueIntf, err := siteConfigTemp.GetSiteConfigFieldValue(key, clusterId, nodeId)

				if err == nil && valueIntf != nil && valueIntf != "" {
					mapIntf[k] = valueIntf
				}
			} else {
				mapIntf[k] = v
			}
		default:
			mapIntf[k] = v
		}
	}

	return mapIntf, nil
}

func translateTemplateKey(key string) (string, error) {
	key = strings.Trim(key, "{ }")
	for search, replace := range map[string]string{
		".Node.":    "siteconfig.Spec.Clusters.Nodes.",
		".Cluster.": "siteconfig.Spec.Clusters.",
		".Site.":    "siteconfig.Spec.",
	} {
		if strings.HasPrefix(key, search) {
			return strings.Replace(key, search, replace, 1), nil
		}
	}
	return "", fmt.Errorf("Key %q could not be translated", key)
}

func populateSpec(filePath string, instantiatedCR map[string]interface{}) error {
	fileData, err := ReadFile(filePath)
	if err != nil {
		return err
	}
	content := make(map[string]string)
	err = yaml.Unmarshal(fileData, content)
	if err != nil {
		return err
	}

	settings := make(map[string]interface{})
	settings["settings"] = content
	instantiatedCR["spec"] = settings
	return nil
}

func (scbuilder *SiteConfigBuilder) getWorkloadManifest(fPath string, cpuSet string, role string) (string, interface{}, error) {

	filePath := fPath + "/" + workloadPath
	crio, err := ReadExtraManifestResourceFile(filePath + "/" + workloadCrioFile)
	if err != nil {
		return "", nil, err
	}
	crioStr := string(crio)
	crioStr = strings.Replace(crioStr, cpuset, cpuSet, -1)
	crioStr = base64.StdEncoding.EncodeToString([]byte(crioStr))

	kubelet, err := ReadExtraManifestResourceFile(filePath + "/" + workloadKubeletFile)
	if err != nil {
		return "", nil, err
	}
	kubeletStr := string(kubelet)
	kubeletStr = strings.Replace(kubeletStr, cpuset, cpuSet, -1)
	kubeletStr = base64.StdEncoding.EncodeToString([]byte(kubeletStr))

	workload, err := ReadExtraManifestResourceFile(filePath + "/" + workloadFile)
	if err != nil {
		return "", nil, err
	}
	workloadStr := string(workload)
	workloadStr = strings.Replace(workloadStr, "$crio", crioStr, -1)
	workloadStr = strings.Replace(workloadStr, "$k8s", kubeletStr, -1)
	workloadStr = strings.Replace(workloadStr, "$mcp", role, -1)

	workloadFileParts := append(strings.Split(workloadFile, "-"), "")
	copy(workloadFileParts[2:], workloadFileParts[1:])
	workloadFileParts[1] = role
	workloadFileForRole := strings.Join(workloadFileParts, "-")

	return workloadFileForRole, reflect.ValueOf(workloadStr).Interface(), nil
}

// getExtraManifestMaps return 2 maps and error
// first map consists: key as fileName, value as file content
// second map contains: key as machineConfigs filename, value as true/false
func (scbuilder *SiteConfigBuilder) getExtraManifestMaps(roles map[string]bool, clusterSpec Clusters, filePath ...string) (map[string]interface{}, map[string]bool, error) {

	var (
		files []fs.FileInfo
		err   error
	)

	dirFiles := &DirContainFiles{}
	dirFilesArray := []DirContainFiles{}
	dataMap := make(map[string]interface{})
	// Manifests to be excluded from merging
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

			filePath := v.Directory + "/" + file.Name()

			if strings.HasSuffix(file.Name(), ".tmpl") {
				// For templates, we can inject the roles directly
				// Assumes that templates that don't care about roles take precautions that they will be called per role.
				for role := range roles {
					filename, value, err := scbuilder.getManifestFromTemplate(filePath, role, clusterSpec)
					if err != nil {
						return dataMap, doNotMerge, err
					}
					if value != "" {
						value, err = addZTPAnnotationToManifest(value)
						if err != nil {
							return dataMap, doNotMerge, err
						}
						dataMap[filename] = value
						// Exclude all templated MCs since they are installation-only MCs
						doNotMerge[filename] = true
					}
				}
			} else {
				// This is a pure passthrough, assuming any static files for both 'master' and 'worker' have their contents set up properly.
				manifestFile, err := ReadExtraManifestResourceFile(filePath)
				if err != nil {
					return dataMap, doNotMerge, err
				}

				manifestFileStr, err := addZTPAnnotationToManifest(string(manifestFile))
				if err != nil {
					return dataMap, doNotMerge, err
				}
				dataMap[file.Name()] = manifestFileStr

			}

		}
	}

	// Adding workload partitions MC only for SNO clusters.
	if clusterSpec.ClusterType == SNO {
		for node := range clusterSpec.Nodes {
			cpuSet := clusterSpec.Nodes[node].Cpuset
			role := clusterSpec.Nodes[node].Role
			if cpuSet != "" {
				//
				for _, v := range dirFilesArray {
					for _, file := range v.Files {
						if file.Name() == workloadPath {
							k, v, err := scbuilder.getWorkloadManifest(v.Directory, cpuSet, role)
							if err != nil {
								errStr := fmt.Sprintf("Error could not read WorkloadManifest %s %s\n", clusterSpec.ClusterName, err)
								return dataMap, doNotMerge, errors.New(errStr)
							} else {
								data, err := addZTPAnnotationToManifest(v.(string))
								if err != nil {
									return dataMap, doNotMerge, err
								}
								dataMap[k] = data
								// Exclude the workload manifest
								doNotMerge[k] = true
							}
						}

					}
				}
			}
		}
	}

	return dataMap, doNotMerge, err
}

func (scbuilder *SiteConfigBuilder) getExtraManifest(dataMap map[string]interface{}, clusterSpec Clusters) (map[string]interface{}, error) {
	// Figure out the list of node roles we need to support in this cluster
	roles := map[string]bool{}
	doNotMerge := map[string]bool{}
	var err error

	for _, node := range clusterSpec.Nodes {
		roles[node.Role] = true
	}

	if clusterSpec.ExtraManifests.SearchPaths != nil {
		dataMap, doNotMerge, err = scbuilder.getExtraManifestMaps(roles, clusterSpec, *clusterSpec.ExtraManifests.SearchPaths...)
		if err != nil {
			return dataMap, err
		}
	} else {
		containerPath, err := GetExtraManifestResourceDir(scbuilder.scBuilderExtraManifestPath)
		if err != nil {
			return dataMap, err
		}
		dataMap, doNotMerge, err = scbuilder.getExtraManifestMaps(roles, clusterSpec, containerPath)
		if err != nil {
			return dataMap, err
		}
	}

	// Adding End User Extra-manifest
	if clusterSpec.ExtraManifests.SearchPaths == nil && clusterSpec.ExtraManifestPath != "" {
		files, err := GetFiles(clusterSpec.ExtraManifestPath)
		if err != nil {
			return dataMap, err
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

			filePath := clusterSpec.ExtraManifestPath + "/" + file.Name()
			manifestFile, err := ReadFile(filePath)
			if err != nil {
				return dataMap, err
			}

			manifestFileStr, err := addZTPAnnotationToManifest(string(manifestFile))
			if err != nil {
				return dataMap, err
			}
			dataMap[file.Name()] = manifestFileStr

			// user provided CRs don't need to be merged
			doNotMerge[file.Name()] = true
		}
	}

	//filter CRs
	dataMap, err = filterExtraManifests(dataMap, clusterSpec.ExtraManifests.Filter)
	if err != nil {
		log.Printf("could not filter %s.%s %s\n", clusterSpec.ClusterName, clusterSpec.ExtraManifestPath, err)
		return dataMap, err
	}

	// check if user wants to merge machineconfigs
	if clusterSpec.MergeDefaultMachineConfigs {
		// merge the pre-defined manifests
		dataMap, err = MergeManifests(dataMap, doNotMerge)
		if err != nil {
			log.Printf("Error could not merge extra-manifest %s.%s %s\n", clusterSpec.ClusterName, clusterSpec.ExtraManifestPath, err)
			return dataMap, err
		}
	}

	return dataMap, nil
}

func (scbuilder *SiteConfigBuilder) getManifestFromTemplate(templatePath, role string, data interface{}) (string, string, error) {
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
		// TODO: The Role should actually be in the data somewhere
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

func (scbuilder *SiteConfigBuilder) splitYamls(yamls []byte) ([][]byte, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(yamls))
	var resources [][]byte

	for {
		var resIntf interface{}
		err := decoder.Decode(&resIntf)

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		resBytes, err := yaml.Marshal(resIntf)

		if err != nil {
			return nil, err
		}
		resources = append(resources, resBytes)
	}

	return resources, nil
}

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
		// in `exclude` more

		// check if include list is empty
		if filter.Exclude != nil && len(filter.Exclude) > 0 {
			errStr := fmt.Sprintf("when InclusionDefault is set to exclude, exclude list can not have entries")
			return dataMap, errors.New(errStr)
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
		if filter.Include != nil && len(filter.Include) > 0 {
			errStr := fmt.Sprintf("when InclusionDefault is set to include, include list can not have entries")
			return dataMap, errors.New(errStr)
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
