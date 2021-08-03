package siteConfig

import (
	"bytes"
	base64 "encoding/base64"
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

func NewSiteConfigBuilder(fileHandler *utils.FilesHandler) *SiteConfigBuilder {
	scBuilder := SiteConfigBuilder{fHandler: fileHandler}
	clusterCRsFile := scBuilder.fHandler.ReadSourceFileCR(clusterCRsFileName)
	clusterCRsYamls, err := scBuilder.splitYamls(clusterCRsFile)
	scBuilder.SourceClusterCRs = make([]interface{}, 9)
	if err != nil {
		panic(err)
	}
	for id, clusterCRsYaml := range clusterCRsYamls {
		var clusterCR interface{}
		err := yaml.Unmarshal(clusterCRsYaml, &clusterCR)

		if err != nil {
			panic(err)
		}
		scBuilder.SourceClusterCRs[id] = clusterCR
	}

	return &scBuilder
}

func (scbuilder *SiteConfigBuilder) Build(siteConfigTemp SiteConfig) map[string][]interface{} {
	clustersCRs := make(map[string][]interface{})

	for id, cluster := range siteConfigTemp.Spec.Clusters {
		if cluster.ClusterName == "" {
			panic("Error: Missing cluster name at site " + siteConfigTemp.Metadata.Name)
		}
		clustersCRs[utils.CustomResource+"/"+siteConfigTemp.Metadata.Name+"/"+cluster.ClusterName] = scbuilder.getClusterCRs(id, siteConfigTemp)
	}

	return clustersCRs
}

func (scbuilder *SiteConfigBuilder) getClusterCRs(clusterId int, siteConfigTemp SiteConfig) []interface{} {
	clusterCRs := make([]interface{}, len(scbuilder.SourceClusterCRs))

	for id, cr := range scbuilder.SourceClusterCRs {
		clusterCRs[id] = scbuilder.getClusterCR(clusterId, siteConfigTemp, cr)
	}

	return clusterCRs
}

func (scbuilder *SiteConfigBuilder) getClusterCR(clusterId int, siteConfigTemp SiteConfig, sourceCR interface{}) interface{} {
	mapIntf := make(map[string]interface{})
	mapSourceCR := sourceCR.(map[string]interface{})

	for k, v := range mapSourceCR {
		if reflect.ValueOf(v).Kind() == reflect.Map {
			mapIntf[k] = scbuilder.getClusterCR(clusterId, siteConfigTemp, v)
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
			dataMap = scbuilder.getExtraManifest(dataMap, cluster)

			// TODO: This should be re-implemented as a template
			if len(cluster.Nodes) > 0 {
				node := siteConfigTemp.Spec.Clusters[0].Nodes[0]
				cpuSet := node.Cpuset
				if cpuSet != "" {
					k, v := scbuilder.getWorkloadManifest(cpuSet)
					dataMap[k] = v
				}
			}
		}

		mapIntf["data"] = dataMap
	}

	return mapIntf
}

func (scbuilder *SiteConfigBuilder) getWorkloadManifest(cpuSet string) (string, interface{}) {
	crio := scbuilder.fHandler.ReadSourceFileCR(workloadPath + "/" + workloadCrioFile)
	crioStr := string(crio)
	crioStr = strings.Replace(crioStr, cpuset, cpuSet, -1)
	crioStr = base64.StdEncoding.EncodeToString([]byte(crioStr))
	kubelet := scbuilder.fHandler.ReadSourceFileCR(workloadPath + "/" + workloadKubeletFile)
	kubeletStr := string(kubelet)
	kubeletStr = strings.Replace(kubeletStr, cpuset, cpuSet, -1)
	kubeletStr = base64.StdEncoding.EncodeToString([]byte(kubeletStr))
	worklod := scbuilder.fHandler.ReadSourceFileCR(workloadPath + "/" + workloadFile)
	workloadStr := string(worklod)
	workloadStr = strings.Replace(workloadStr, "$crio", crioStr, -1)
	workloadStr = strings.Replace(workloadStr, "$k8s", kubeletStr, -1)

	return workloadFile, reflect.ValueOf(workloadStr).Interface()
}

func (scbuilder *SiteConfigBuilder) getExtraManifest(dataMap map[string]interface{}, clusterSpec Clusters) map[string]interface{} {

	for _, file := range scbuilder.fHandler.GetSourceFiles(extraManifestPath) {
		filePath := extraManifestPath + "/" + file.Name()
		if strings.HasSuffix(file.Name(), ".tmpl") {
			// FIXME: Hard-coding "master" as the role is only valid for SNO -
			// In the future we should run this multiple times, one for each
			// role
			filename, value := scbuilder.getManifestFromTemplate(filePath, "master", clusterSpec)
			if value != "" {
				dataMap[filename] = value
			}
		} else {
			manifestFile := scbuilder.fHandler.ReadSourceFileCR(filePath)
			manifestFileStr := string(manifestFile)
			dataMap[file.Name()] = manifestFileStr
		}
	}
	return dataMap
}

func (scbuilder *SiteConfigBuilder) getManifestFromTemplate(templatePath, role string, data interface{}) (string, string) {
	baseName := filepath.Base(templatePath)
	renderedName := fmt.Sprintf("%s-%s", role, strings.TrimSuffix(baseName, ".tmpl"))
	tStr := scbuilder.fHandler.ReadSourceFileCR(templatePath)
	t, err := template.New(baseName).Parse(string(tStr))
	if err != nil {
		return "", ""
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
		return "", ""
	}
	// Ensure there's non-whitespace content
	for _, r := range output.String() {
		if !unicode.IsSpace(r) {
			return renderedName, output.String()
		}
	}
	// Output is all whitespace; return nil instead
	return "", ""
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
