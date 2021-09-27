package siteConfig

import (
	"bytes"
	base64 "encoding/base64"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
	"unicode"

	utils "github.com/openshift-kni/cnf-features-deploy/ztp/ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator/utils"
	yaml "gopkg.in/yaml.v3"
)

type SiteConfigBuilder struct {
	fHandler         *utils.FilesHandler
	SourceClusterCRs []interface{}
}

func NewSiteConfigBuilder(fileHandler *utils.FilesHandler) (*SiteConfigBuilder, error) {
	scBuilder := SiteConfigBuilder{fHandler: fileHandler}
	clusterCRsFile, err := scBuilder.fHandler.ReadResourceFile(clusterCRsFileName)
	if err != nil {
		return &scBuilder, err
	}

	clusterCRsYamls, err := scBuilder.splitYamls(clusterCRsFile)
	if err != nil {
		return &scBuilder, err
	}
	scBuilder.SourceClusterCRs = make([]interface{}, 9)
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

func (scbuilder *SiteConfigBuilder) Build(siteConfigTemp SiteConfig) (map[string][]interface{}, error) {
	clustersCRs := make(map[string][]interface{})

	for id, cluster := range siteConfigTemp.Spec.Clusters {
		if cluster.ClusterName == "" {
			return clustersCRs, errors.New("Error: Missing cluster name at site " + siteConfigTemp.Metadata.Name)
		}
		if cluster.NetworkType == "" {
			cluster.NetworkType = "OVNKubernetes"
			siteConfigTemp.Spec.Clusters[id].NetworkType = "OVNKubernetes"
		}
		if cluster.NetworkType != "OpenShiftSDN" && cluster.NetworkType != "OVNKubernetes" {
			return clustersCRs, errors.New("Error: networkType must be either OpenShiftSDN or OVNKubernetes " + siteConfigTemp.Metadata.Name)
		}
		siteConfigTemp.Spec.Clusters[id].NetworkType = "{\"networking\":{\"networkType\":\"" + cluster.NetworkType + "\"}}"
		clusterValue, err := scbuilder.getClusterCRs(id, siteConfigTemp)
		if err != nil {
			return clustersCRs, err
		}
		clustersCRs[utils.CustomResource+"/"+siteConfigTemp.Metadata.Name+"/"+cluster.ClusterName] = clusterValue
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
				return clusterCRs, err
			}

			// FIXME: Assuming 1 node for SNO deployment needs to be changed for RWN deployment
			if len(siteConfigTemp.Spec.Clusters[clusterId].Nodes) > 0 {
				cpuSet := siteConfigTemp.Spec.Clusters[clusterId].Nodes[0].Cpuset
				if cpuSet != "" {
					k, v, err := scbuilder.getWorkloadManifest(cpuSet)
					if err != nil {
						return clusterCRs, err
					}
					dataMap[k] = v
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
	crio, err := scbuilder.fHandler.ReadSourceFile(workloadPath + "/" + workloadCrioFile)
	if err != nil {
		return "", nil, err
	}
	crioStr := string(crio)
	crioStr = strings.Replace(crioStr, cpuset, cpuSet, -1)
	crioStr = base64.StdEncoding.EncodeToString([]byte(crioStr))
	kubelet, err := scbuilder.fHandler.ReadSourceFile(workloadPath + "/" + workloadKubeletFile)
	if err != nil {
		return "", nil, err
	}
	kubeletStr := string(kubelet)
	kubeletStr = strings.Replace(kubeletStr, cpuset, cpuSet, -1)
	kubeletStr = base64.StdEncoding.EncodeToString([]byte(kubeletStr))
	worklod, err := scbuilder.fHandler.ReadSourceFile(workloadPath + "/" + workloadFile)
	if err != nil {
		return "", nil, err
	}
	workloadStr := string(worklod)
	workloadStr = strings.Replace(workloadStr, "$crio", crioStr, -1)
	workloadStr = strings.Replace(workloadStr, "$k8s", kubeletStr, -1)

	return workloadFile, reflect.ValueOf(workloadStr).Interface(), nil
}

func (scbuilder *SiteConfigBuilder) getExtraManifest(dataMap map[string]interface{}, clusterSpec Clusters) (map[string]interface{}, error) {
	files, err := scbuilder.fHandler.GetSourceFiles(extraManifestPath)

	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := extraManifestPath + "/" + file.Name()
		if strings.HasSuffix(file.Name(), ".tmpl") {
			// FIXME: Hard-coding "master" as the role is only valid for SNO -
			// In the future we should run this multiple times, one for each
			// role
			filename, value, err := scbuilder.getManifestFromTemplate(filePath, "master", clusterSpec)
			if err != nil {
				return dataMap, err
			}
			if value != "" {
				dataMap[filename] = value
			}
		} else {
			manifestFile, err := scbuilder.fHandler.ReadSourceFile(filePath)
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
	tStr, err := scbuilder.fHandler.ReadSourceFile(templatePath)
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
