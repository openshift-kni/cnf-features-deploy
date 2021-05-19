package siteConfig

import (
	"bytes"
	"io"
	utils "github.com/openshift-kni/cnf-features-deploy/ztp/ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator/utils"
	yaml "gopkg.in/yaml.v3"
	"strings"
	"reflect"
	//"fmt"
)

type SiteConfigBuilder struct {
	fHandler *utils.FilesHandler
	SourceClusterCRs []interface{}
}

func NewSiteConfigBuilder(fileHandler *utils.FilesHandler) *SiteConfigBuilder {
	scBuilder := SiteConfigBuilder{fHandler:fileHandler}
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

func (scbuilder *SiteConfigBuilder) Build(siteConfigTemp SiteConfig) (map[string][]interface{}) {
	clustersCRs := make(map[string][]interface{})

	for id, cluster:= range siteConfigTemp.Spec.Clusters {
		clustersCRs[utils.CustomResource + "/" + siteConfigTemp.Metadata.Name + "/" + cluster.ClusterName] = scbuilder.getClusterCRs(id, siteConfigTemp)
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
	mapIntf := sourceCR.(map[string]interface{})

	for k, v := range mapIntf {
		if reflect.ValueOf(v).Kind() == reflect.Map {
			mapIntf[k] = scbuilder.getClusterCR(clusterId, siteConfigTemp, v)
		} else if reflect.ValueOf(v).Kind() == reflect.String &&
			strings.HasPrefix(v.(string), "siteconfig.") {
			valueIntf, err := siteConfigTemp.GetSiteConfigFieldValue(v.(string), clusterId, 0)
			if err != nil || valueIntf == nil {
				delete(mapIntf, k)
			} else {
				mapIntf[k] = valueIntf
			}
		}
	}

	return mapIntf
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
