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
		clusterValue, err := scbuilder.getClusterCRs(id, siteConfigTemp)
		if err != nil {
			return clustersCRs, err
		}

		clustersCRs[utils.CustomResource+"/"+siteConfigTemp.Metadata.Name+"/"+cluster.ClusterName] = clusterValue
	}

	return clustersCRs, nil
}

func (scbuilder *SiteConfigBuilder) getClusterCRs(clusterId int, siteConfigTemp SiteConfig) ([]interface{}, error) {
	clusterCRs := make([]interface{}, len(scbuilder.SourceClusterCRs))

	for id, cr := range scbuilder.SourceClusterCRs {
		crValue, err := scbuilder.getClusterCR(clusterId, siteConfigTemp, cr)
		if err != nil {
			return clusterCRs, err
		}
		clusterCRs[id] = crValue
	}


	return clusterCRs, nil
}

func (scbuilder *SiteConfigBuilder) getClusterCR(clusterId int, siteConfigTemp SiteConfig, sourceCR interface{}) (interface{}, error) {
	mapIntf := make(map[string]interface{})
	mapSourceCR := sourceCR.(map[string]interface{})

	for k, v := range mapSourceCR {
		if (k == "controlPlaneAgents") {
			numMasters:=0

			if len(siteConfigTemp.Spec.Clusters) > 0 {
				cluster := siteConfigTemp.Spec.Clusters[0]
				if (cluster.ClusterType == "sno") {
					numMasters = 1
				} else if  (cluster.ClusterType == "standard") {
					numMasters = 3
				}
			}

			mapIntf[k] = numMasters
		} else if reflect.ValueOf(v).Kind() == reflect.Map {
			value, err := scbuilder.getClusterCR(clusterId, siteConfigTemp, v)
			if err != nil {
				return mapIntf, err
			}

			mapIntf[k] = value //scbuilder.getClusterCR(clusterId, siteConfigTemp, v)
		} else if reflect.ValueOf(v).Kind() == reflect.String &&
			strings.HasPrefix(v.(string), "siteconfig.") {
			valueIntf, err := siteConfigTemp.GetSiteConfigFieldValue(v.(string), clusterId, 0)

			if err == nil && valueIntf != nil && valueIntf != "" {
				mapIntf[k] = valueIntf
			}
		} else {
			mapIntf[k] = v
		}
	}


	// Adding extra manifest
	if mapSourceCR["kind"] == "ConfigMap" {
		dataMap := make(map[string]interface{})
		// FIXME: Assuming 1 cluster and 1 node for SNO deployment needs to be changed for RWN deployment
		if len(siteConfigTemp.Spec.Clusters) > 0 {
			cluster := siteConfigTemp.Spec.Clusters[0]
			dataMap, err := scbuilder.getExtraManifest(dataMap, cluster)
			if err != nil {
				return dataMap, err
			}

			// TODO: This should be re-implemented as a template
			if len(cluster.Nodes) > 0 {
				node := siteConfigTemp.Spec.Clusters[0].Nodes[0]
				cpuSet := node.Cpuset
				if cpuSet != "" {
					k, v, err := scbuilder.getWorkloadManifest(cpuSet)
					if err != nil {
						return mapIntf, err
					}
					dataMap[k] = v
				}
			}
		}

		mapIntf["data"] = dataMap
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
