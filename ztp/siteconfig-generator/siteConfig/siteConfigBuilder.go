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

	for id, cluster := range siteConfigTemp.Spec.Clusters {
		if cluster.ClusterName == "" {
			return clustersCRs, errors.New("Error: Missing cluster name at site " + siteConfigTemp.Metadata.Name)
		}
		if cluster.NetworkType != "OpenShiftSDN" && cluster.NetworkType != "OVNKubernetes" {
			return clustersCRs, errors.New("Error: networkType must be either OpenShiftSDN or OVNKubernetes " + siteConfigTemp.Metadata.Name + "/" + cluster.ClusterName)
		}
		if clustersCRs[siteConfigTemp.Metadata.Name+"/"+cluster.ClusterName] != nil {
			return clustersCRs, errors.New("Error: Repeated Cluster Name " + siteConfigTemp.Metadata.Name + "/" + cluster.ClusterName)
		}
		siteConfigTemp.Spec.Clusters[id].NetworkType = "{\"networking\":{\"networkType\":\"" + cluster.NetworkType + "\"}}"
		clusterCRs, err := scbuilder.getClusterCRs(id, siteConfigTemp)
		if err != nil {
			return clustersCRs, err
		}
		clustersCRs[siteConfigTemp.Metadata.Name+"/"+cluster.ClusterName] = clusterCRs
	}

	return clustersCRs, nil
}

func (scbuilder *SiteConfigBuilder) getClusterCRs(clusterId int, siteConfigTemp SiteConfig) ([]interface{}, error) {
	var clusterCRs []interface{}

	for _, cr := range scbuilder.SourceClusterCRs {
		mapSourceCR := cr.(map[string]interface{})

		if mapSourceCR["kind"] == "ConfigMap" {
			dataMap := make(map[string]interface{})
			cluster := siteConfigTemp.Spec.Clusters[clusterId]
			dataMap, err := scbuilder.getExtraManifest(dataMap, cluster)
			if err != nil {
				// Will return and fail if the end user extra-manifest having issues.
				log.Printf("Error could not create extra-manifest %s.%s %s\n", cluster.ClusterName, cluster.ExtraManifestPath, err)
				return clusterCRs, err
			}

			// Adding workload partitions MC only for SNO clusters.
			if siteConfigTemp.Spec.Clusters[clusterId].ClusterType == SNO &&
				len(siteConfigTemp.Spec.Clusters[clusterId].Nodes) > 0 {
				cpuSet := siteConfigTemp.Spec.Clusters[clusterId].Nodes[0].Cpuset
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
			crValue, err := scbuilder.getClusterCR(clusterId, siteConfigTemp, mapSourceCR, -1)
			if err != nil {
				return clusterCRs, err
			}
			clusterCRs = append(clusterCRs, crValue)
		} else if mapSourceCR["kind"] == "BareMetalHost" || mapSourceCR["kind"] == "NMStateConfig" {
			for ndId := range siteConfigTemp.Spec.Clusters[clusterId].Nodes {
				crValue, err := scbuilder.getClusterCR(clusterId, siteConfigTemp, mapSourceCR, ndId)
				if err != nil {
					return clusterCRs, err
				}
				clusterCRs = append(clusterCRs, crValue)
			}
		} else {
			crValue, err := scbuilder.getClusterCR(clusterId, siteConfigTemp, mapSourceCR, -1)
			if err != nil {
				return clusterCRs, err
			}
			clusterCRs = append(clusterCRs, crValue)
		}
	}

	return clusterCRs, nil
}

func (scbuilder *SiteConfigBuilder) getClusterCR(clusterId int, siteConfigTemp SiteConfig, mapSourceCR map[string]interface{}, nodeId int) (interface{}, error) {
	mapIntf := make(map[string]interface{})

	for k, v := range mapSourceCR {
		if reflect.ValueOf(v).Kind() == reflect.Map {
			value, err := scbuilder.getClusterCR(clusterId, siteConfigTemp, v.(map[string]interface{}), nodeId)
			if err != nil {
				return mapIntf, err
			}
			mapIntf[k] = value
		} else if reflect.ValueOf(v).Kind() == reflect.String &&
			strings.HasPrefix(v.(string), "siteconfig.") {
			valueIntf, err := siteConfigTemp.GetSiteConfigFieldValue(v.(string), clusterId, nodeId)

			if err == nil && valueIntf != nil && valueIntf != "" {
				mapIntf[k] = valueIntf
			}
		} else {
			mapIntf[k] = v
		}
	}

	return mapIntf, nil
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
