package policyGen

import (
	"io/ioutil"
	utils "github.com/serngawy/cnf-features-deploy/ztp/ztp-ran-policy-generator/kustomize/plugin/ranPolicyGenerator/v1/ranpolicygenerator/utils"
	//"fmt"
	//"bytes"
	//"io"
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

func (pbuilder *PolicyBuilder) Build() (map[string]interface{}) {

	policies := make(map[string]interface{})
	if len(pbuilder.RanGenTemp.SourceFiles) != 0 {
		namespace, path, matchKey, matchValue, matchOper := pbuilder.getPolicyNsPath()
		subjects := make([]utils.Subject , 0)
		for id, sFile := range pbuilder.RanGenTemp.SourceFiles {
			pname, rname := pbuilder.getPolicyName(id)
			sPolicyFile, err := ioutil.ReadFile(pbuilder.SourcePoliciesDir + "/" + sFile.FileName + utils.FileExt)
			if err != nil {
				panic(err)
			}
			rname, resourceDef := pbuilder.getResourceDefinition(sFile.Spec, sPolicyFile, rname, pbuilder.RanGenTemp.Metadata.Labels.Mcp)

			acmPolicy := pbuilder.getPolicy(pname + "-" + rname, namespace, resourceDef)
			policies[path + "/" + pname + "-" + rname] = acmPolicy
			subject := CreatePolicySubject(pname + "-" + rname )
			subjects = append(subjects, subject)
		}

		placementRule := CreatePlacementRule(pbuilder.RanGenTemp.Metadata.Name, namespace, matchKey, matchOper, matchValue)
		policies[path + "/" + placementRule.Metadata.Name] = placementRule

		placementBinding := CreatePlacementBinding(pbuilder.RanGenTemp.Metadata.Name, namespace, placementRule.Metadata.Name, subjects)
		policies[path + "/" + placementBinding.Metadata.Name] = placementBinding
	}
	return policies
}

func (pbuilder *PolicyBuilder) getPolicy(name string, namespace string, objMap map[string]interface{}) utils.AcmPolicy {
	objTemp := CreateObjTemplates(objMap)
	acmConfigPolicy := CreateAcmConfigPolicy(name)

	objTempArr := make([]utils.ObjectTemplates, 1)
	objTempArr[0] = objTemp
	acmConfigPolicy.Spec.ObjectTemplates = objTempArr

	policyObjDef := utils.PolicyObjectDefinition{}
	policyObjDef.ObjDef = acmConfigPolicy

	acmPolicy := CreateAcmPolicy(name, namespace)
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

	// Get resource name from source policy if name is empty or N/A
	if name == "" || name == utils.NotApplicable {
		name = sourcePolicyMap["metadata"].(map[string]interface{})["name"].(string)
	}
	if len(spec) != 0 {
		sourcePolicyMap["spec"] = pbuilder.setSpecValues(sourcePolicyMap["spec"].(map[string]interface{}), spec)
	}
	if (sourcePolicyMap["spec"] != nil && len(spec) != sourcePolicyMap["spec"]) {
		sourcePolicyMap["spec"] = pbuilder.removeUnsetValues(sourcePolicyMap["spec"].(map[string]interface{}))
	}

	return name, sourcePolicyMap
}

func (pbuilder *PolicyBuilder) removeUnsetValues(spec map[string]interface{}) map[string]interface{} {
	for k, v := range spec {
		if reflect.ValueOf(spec[k]).Kind() == reflect.Map {
			spec[k] = pbuilder.removeUnsetValues(spec[k].(map[string]interface{}))
		} else if reflect.ValueOf(spec[k]).Kind() == reflect.String {
			if strings.HasPrefix(v.(string), "$") {
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

func (pbuilder *PolicyBuilder) getPolicyNsPath() (string , string, string, string, string) {
	ns := ""
	path := ""
	matchKey := ""
	matchOper := ""
	matchValue := ""
	if pbuilder.RanGenTemp.Metadata.Name != "" {
		if pbuilder.RanGenTemp.Metadata.Labels.SiteName != utils.NotApplicable {
			ns = utils.SiteNS
			matchKey = utils.Sites
			matchOper = utils.InOper
			matchValue = pbuilder.RanGenTemp.Metadata.Labels.SiteName
			path = utils.Sites + "/" + pbuilder.RanGenTemp.Metadata.Labels.SiteName
		} else if pbuilder.RanGenTemp.Metadata.Labels.GroupName != utils.NotApplicable {
			ns = utils.GroupNS
			matchKey = pbuilder.RanGenTemp.Metadata.Labels.GroupName
			matchOper = utils.ExistOper
			path = utils.Groups + "/" + pbuilder.RanGenTemp.Metadata.Labels.GroupName
		} else if pbuilder.RanGenTemp.Metadata.Labels.Common {
			ns = utils.CommonNS
			matchKey = utils.Common
			matchOper = utils.InOper
			matchValue = "true"
			path = utils.Common
		} else {
			panic("Error: missing metadata info either siteName, groupName or common should be set")
		}
	}
	return ns, path, matchKey, matchValue, matchOper
}

func (pbuilder *PolicyBuilder) getPolicyName(sFileId int) (string , string) {
	pname := ""
	rname := ""
	if pbuilder.RanGenTemp.Metadata.Name != "" {
		if pbuilder.RanGenTemp.Metadata.Labels.SiteName != utils.NotApplicable {
			pname = pbuilder.RanGenTemp.Metadata.Labels.SiteName
		} else if pbuilder.RanGenTemp.Metadata.Labels.GroupName != utils.NotApplicable {
			pname = pbuilder.RanGenTemp.Metadata.Labels.GroupName
		} else if pbuilder.RanGenTemp.Metadata.Labels.Common {
			pname = utils.Common
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
	// The names in the yaml must be compliant RFC 1123 domain names (all lower case)
	pname = strings.ToLower(pname)
	rname = strings.ToLower(rname)
	return pname, rname
}
