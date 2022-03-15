package siteConfig

import (
	"bytes"
	base64 "encoding/base64"
	"errors"
	"fmt"
	"io"
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

	clusterCRsYamls, err := scBuilder.splitYamls([]byte(clusterCRs))
	if err != nil {
		return &scBuilder, err
	}
	scBuilder.SourceClusterCRs = make([]interface{}, len(clusterCRsYamls))
	for id, clusterCRsYaml := range clusterCRsYamls {
		var clusterCR interface{}
		err := yaml.Unmarshal(clusterCRsYaml, &clusterCR)

		if err != nil {
			return &scBuilder, err
		}
		scBuilder.SourceClusterCRs[id] = clusterCR
	}

	return &scBuilder, nil
}

func (scbuilder *SiteConfigBuilder) SetLocalExtraManifestPath(path string) {
	scbuilder.scBuilderExtraManifestPath = path
}

func (scbuilder *SiteConfigBuilder) Build(siteConfigTemp SiteConfig) (map[string][]interface{}, error) {
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
		if cluster.NetworkType != "OpenShiftSDN" && cluster.NetworkType != "OVNKubernetes" {
			return errors.New("Error: networkType must be either OpenShiftSDN or OVNKubernetes " + siteConfigTemp.Metadata.Name + "/" + cluster.ClusterName)
		}

		siteConfigTemp.Spec.Clusters[id].NetworkType = "{\"networking\":{\"networkType\":\"" + cluster.NetworkType + "\"}}"

		if siteConfigTemp.Spec.ClusterImageSetNameRef == "" && siteConfigTemp.Spec.Clusters[id].ClusterImageSetNameRef == "" {
			return errors.New("Error: Site and cluster clusterImageSetNameRef cannot be empty " + siteConfigTemp.Metadata.Name + "/" + cluster.ClusterName)
		}
		// If cluster has not set a clusterImageSetNameRef we use the site one
		if siteConfigTemp.Spec.Clusters[id].ClusterImageSetNameRef == "" {
			siteConfigTemp.Spec.Clusters[id].ClusterImageSetNameRef = siteConfigTemp.Spec.ClusterImageSetNameRef
		}

		clusters[siteConfigTemp.Metadata.Name+"/"+cluster.ClusterName] = true
	}
	return nil
}

func (scbuilder *SiteConfigBuilder) getClusterCRs(clusterId int, siteConfigTemp SiteConfig) ([]interface{}, error) {
	var clusterCRs []interface{}
	validKinds := make(map[string]bool)
	validNodeKinds := make(map[string]bool)

	for _, cr := range scbuilder.SourceClusterCRs {
		mapSourceCR := cr.(map[string]interface{})
		kind := mapSourceCR["kind"].(string)
		validKinds[kind] = true
		cluster := siteConfigTemp.Spec.Clusters[clusterId]

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
					clusterCRs = append(clusterCRs, instantiatedCR)
				}
			}
		} else {
			// cluster-level CR
			if kind == "ConfigMap" {
				// For ConfigMap, add all the ExtraManifest files to it before doing further instantiation:
				dataMap := make(map[string]interface{})
				dataMap, err := scbuilder.getExtraManifest(dataMap, cluster)
				if err != nil {
					// Will return and fail if the end user extra-manifest having issues.
					log.Printf("Error could not create extra-manifest %s.%s %s\n", cluster.ClusterName, cluster.ExtraManifestPath, err)
					return clusterCRs, err
				}

				// Adding workload partitions MC only for SNO clusters.
				if cluster.ClusterType == SNO &&
					len(cluster.Nodes) > 0 {
					cpuSet := cluster.Nodes[0].Cpuset
					if cpuSet != "" {
						k, v, err := scbuilder.getWorkloadManifest(cpuSet)
						if err != nil {
							log.Printf("Error could not read WorkloadManifest %s %s\n", cluster.ClusterName, err)
							return clusterCRs, err
						} else {
							dataMap[k] = v
						}
					}
				}

				mapSourceCR["data"] = dataMap
			}

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

			clusterCRs = append(clusterCRs, instantiatedCR)
		}
	}

	// Double-check that the user didn't ask us to override any invalid object types:
	err := siteConfigTemp.areAllOverridesValid(&validKinds, &validNodeKinds)
	if err != nil {
		return clusterCRs, err
	}

	return clusterCRs, nil
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
	overriddenMetadata := overriddenCR["metadata"].(map[string]interface{})
	for _, field := range []string{"name", "namespace"} {
		if originalMetadata[field] != overriddenMetadata[field] {
			return overriddenCR, fmt.Errorf("Overridden template metadata.%s %q does not match expected value %q", field, overriddenMetadata[field], originalMetadata[field])
		}
	}
	originalAnnotations := originalMetadata["annotations"].(map[string]interface{})
	overriddenAnnotations := overriddenMetadata["annotations"].(map[string]interface{})
	argocdAnnotation := "argocd.argoproj.io/sync-wave"
	if originalAnnotations[argocdAnnotation] != overriddenAnnotations[argocdAnnotation] {
		return overriddenCR, fmt.Errorf("Overridden template metadata.annotations[%q] %q does not match expected value %q", argocdAnnotation, overriddenAnnotations[argocdAnnotation], originalAnnotations[argocdAnnotation])
	}
	return overriddenCR, nil
}

func (scbuilder *SiteConfigBuilder) getClusterCR(clusterId int, siteConfigTemp SiteConfig, mapSourceCR map[string]interface{}, nodeId int) (map[string]interface{}, error) {
	mapIntf := make(map[string]interface{})

	for k, v := range mapSourceCR {
		if reflect.ValueOf(v).Kind() == reflect.Map {
			value, err := scbuilder.getClusterCR(clusterId, siteConfigTemp, v.(map[string]interface{}), nodeId)
			if err != nil {
				return mapIntf, err
			}
			mapIntf[k] = value
		} else if reflect.ValueOf(v).Kind() == reflect.String &&
			strings.HasPrefix(v.(string), "{{") &&
			strings.HasSuffix(v.(string), "}}") {
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

func (scbuilder *SiteConfigBuilder) getWorkloadManifest(cpuSet string) (string, interface{}, error) {
	filePath := scbuilder.scBuilderExtraManifestPath + "/" + workloadPath
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
	worklod, err := ReadExtraManifestResourceFile(filePath + "/" + workloadFile)
	if err != nil {
		return "", nil, err
	}
	workloadStr := string(worklod)
	workloadStr = strings.Replace(workloadStr, "$crio", crioStr, -1)
	workloadStr = strings.Replace(workloadStr, "$k8s", kubeletStr, -1)

	return workloadFile, reflect.ValueOf(workloadStr).Interface(), nil
}

func (scbuilder *SiteConfigBuilder) getExtraManifest(dataMap map[string]interface{}, clusterSpec Clusters) (map[string]interface{}, error) {
	// Figure out the list of node roles we need to support in this cluster
	roles := map[string]bool{}
	for _, node := range clusterSpec.Nodes {
		roles[node.Role] = true
	}

	// Adding the pre-defined DU profile extra-manifest.
	files, err := GetExtraManifestResourceFiles(scbuilder.scBuilderExtraManifestPath)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() || file.Name()[0] == '.' {
			continue
		}

		filePath := scbuilder.scBuilderExtraManifestPath + "/" + file.Name()
		if strings.HasSuffix(file.Name(), ".tmpl") {
			// For templates, we can inject the roles directly
			// Assumes that templates that don't care about roles take precautions that they will be called per role.
			for role := range roles {
				filename, value, err := scbuilder.getManifestFromTemplate(filePath, role, clusterSpec)
				if err != nil {
					return dataMap, err
				}
				if value != "" {
					dataMap[filename] = value
				}
			}
		} else {
			// This is a pure passthrough, assuming any static files for both 'master' and 'worker' have their contents set up properly.
			manifestFile, err := ReadExtraManifestResourceFile(filePath)
			if err != nil {
				return dataMap, err
			}

			manifestFileStr := string(manifestFile)
			dataMap[file.Name()] = manifestFileStr
		}
	}

	// Adding End User Extra-manifest
	if clusterSpec.ExtraManifestPath != "" {
		files, err = GetFiles(clusterSpec.ExtraManifestPath)
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

			manifestFileStr := string(manifestFile)
			dataMap[file.Name()] = manifestFileStr
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
