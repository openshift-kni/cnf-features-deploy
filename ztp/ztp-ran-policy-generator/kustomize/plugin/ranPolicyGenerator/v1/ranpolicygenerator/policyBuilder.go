package main

import (
	"io/ioutil"
	utils "github.com/cnf-features-deploy/ztp/ztp-ran-policy-generator/kustomize/plugin/ranPolicyGenerator/v1/ranpolicygenerator/utils"
	//"fmt"
	yaml "gopkg.in/yaml.v3"
	"strings"
	"reflect"
)

type PolicyBuilder struct {
	RanGenTemp utils.RanGenTemplate
	SourcePoliciesDir string
}

func NewPolicyBuilder(RanGenTemp utils.RanGenTemplate, SourcePoliciesDir string) *PolicyBuilder {
	return &PolicyBuilder{RanGenTemp:RanGenTemp, SourcePoliciesDir:SourcePoliciesDir}
}

func (pbuilder *PolicyBuilder) build() map[string]utils.AcmPolicy {

	policies := make(map[string]utils.AcmPolicy)
	if len(pbuilder.RanGenTemp.SourceFiles) != 0 {
		for id, sFile := range pbuilder.RanGenTemp.SourceFiles {
			pname, rname, namespace := pbuilder.getPolicyNameNS(id)
			sPolicyFile, _ := ioutil.ReadFile(pbuilder.SourcePoliciesDir + "/" + sFile.FileName + utils.FileExt)
			rname, resourceDef := pbuilder.getResourceDefinition(sFile.Spec, sPolicyFile, rname, pbuilder.RanGenTemp.Metadata.Labels.Mcp)

			acmPolicy := pbuilder.getPolicy(pname + "-" + rname, namespace, resourceDef)
			policies[pname + "-" + rname] = acmPolicy
		}
	}
	return policies
}

func (pbuilder *PolicyBuilder) getPolicy(name string, namespace string, objMap map[string]interface{}) utils.AcmPolicy {
	objTemp := utils.CreateObjTemplates(objMap)
	acmConfigPolicy := utils.CreateAcmConfigPolicy(name)

	objTempArr := make([]utils.ObjectTemplates, 1)
	objTempArr[0] = objTemp
	acmConfigPolicy.Spec.ObjectTemplates = objTempArr

	acmConfigPolicyArr := make([]utils.AcmConfigurationPolicy, 1)
	acmConfigPolicyArr[0] = acmConfigPolicy

	policyObjDef := utils.PolicyObjectDefinition{}
	policyObjDef.ObjDef = acmConfigPolicyArr

	acmPolicy := utils.CreateAcmPolicy(name, namespace)
	policyObjDefArr := make([]utils.PolicyObjectDefinition, 1)
	policyObjDefArr[0] = policyObjDef

	acmPolicy.Spec.PolicyTemplates = policyObjDefArr
	return acmPolicy
}

func (pbuilder *PolicyBuilder) getResourceDefinition(spec map[string]interface{}, sourcePolicy []byte, name string, mcp string) (string, map[string]interface{}) {
	sourcePolicyMap := make(map[string]interface{})
	sourcePolicyStr := string(sourcePolicy)
	if name != "" && name != utils.NotApplicable {
		sourcePolicyStr = strings.Replace(sourcePolicyStr, "$name", name, -1)
	}
	if mcp != "" && mcp != utils.NotApplicable {
		sourcePolicyStr = strings.Replace(sourcePolicyStr, "$mcp", mcp, -1)
	}

	err := yaml.Unmarshal([]byte(sourcePolicyStr), &sourcePolicyMap)
	if err != nil {
		panic(err)
	}
	// Get policy name from source policy if name is empty or N/A
	if name == "" || name == utils.NotApplicable {
		name = sourcePolicyMap["metadata"].(map[string]interface{})["name"].(string)
	}
	if len(spec) != 0 {
		sourcePolicyMap["spec"] = pbuilder.setSpecValues(sourcePolicyMap["spec"].(map[string]interface{}), spec)
		sourcePolicyMap["spec"] = pbuilder.removeUnsetValues(sourcePolicyMap["spec"].(map[string]interface{}))
	}

	return name, sourcePolicyMap
}

func (pbuilder *PolicyBuilder) removeUnsetValues(spec map[string]interface{}) map[string]interface{} {
	for k, v := range spec {
		if reflect.ValueOf(spec[k]).Kind() == reflect.Map {
			spec[k] = pbuilder.removeUnsetValues(spec[k].(map[string]interface{}))
		} else if reflect.ValueOf(spec[k]).Kind() == reflect.String {
			if strings.Contains(v.(string), "$") {
				delete(spec, k)
			}
		}
	}
	return spec
}

func (pbuilder *PolicyBuilder) setSpecValues(sourceMap map[string]interface{}, valueMap map[string]interface{}) map[string]interface{} {
	for k, v := range valueMap {
		if reflect.ValueOf(sourceMap[k]).Kind() == reflect.Map {
			sourceMap[k] = pbuilder.setSpecValues(sourceMap[k].(map[string]interface{}),v.(map[string]interface{}))
		} else {
			sourceMap[k] = v
		}
	}
	return sourceMap
}

func (pbuilder *PolicyBuilder) getPolicyNameNS(sFileId int) (string , string, string) {
	pname := ""
	rname := ""
	ns := ""
	if pbuilder.RanGenTemp.Metadata.Name != "" {
		if pbuilder.RanGenTemp.Metadata.Labels.SiteName != utils.NotApplicable {
			pname = pbuilder.RanGenTemp.Metadata.Labels.SiteName
			ns = utils.SiteNS
		} else if pbuilder.RanGenTemp.Metadata.Labels.GroupName != utils.NotApplicable {
			pname = pbuilder.RanGenTemp.Metadata.Labels.GroupName
			ns = utils.GroupNS
		} else if pbuilder.RanGenTemp.Metadata.Labels.Common {
			pname = "common"
			ns = utils.CommonNS
		} else {
			panic("Error: missing metadata info either siteName, groupName or common should be set")
		}
		if len(pbuilder.RanGenTemp.SourceFiles) > sFileId {
			if pbuilder.RanGenTemp.SourceFiles[sFileId].FileName != "" {
				pname = pname + "-" + pbuilder.RanGenTemp.SourceFiles[sFileId].FileName
			}
			if pbuilder.RanGenTemp.SourceFiles[sFileId].Name != utils.NotApplicable &&
				pbuilder.RanGenTemp.SourceFiles[sFileId].Name != ""{
					rname = pbuilder.RanGenTemp.SourceFiles[sFileId].Name
			}
		}
	}
	return pname, rname, ns
}