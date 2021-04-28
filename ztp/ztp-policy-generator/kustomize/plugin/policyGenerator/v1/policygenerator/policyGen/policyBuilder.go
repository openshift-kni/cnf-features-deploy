package policyGen

import (
	"io/ioutil"
	utils "github.com/openshift-kni/cnf-features-deploy/ztp/ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator/utils"
	//"fmt"
	yaml "gopkg.in/yaml.v3"
	"strings"
	"reflect"
)

type PolicyBuilder struct {
	PolicyGenTemp utils.PolicyGenTemplate
	SourcePoliciesDir string
}

func NewPolicyBuilder(PolicyGenTemp utils.PolicyGenTemplate, SourcePoliciesDir string) *PolicyBuilder {
	return &PolicyBuilder{PolicyGenTemp:PolicyGenTemp, SourcePoliciesDir:SourcePoliciesDir}
}

func (pbuilder *PolicyBuilder) Build(customResourseOnly bool) (map[string]interface{}) {

	policies := make(map[string]interface{})
	if len(pbuilder.PolicyGenTemp.SourceFiles) != 0 && !customResourseOnly {
		namespace, path, matchKey, matchValue, matchOper := pbuilder.getPolicyNsPath()
		subjects := make([]utils.Subject , 0)
		for id, sFile := range pbuilder.PolicyGenTemp.SourceFiles {
			pname, rname := pbuilder.getPolicyName(id)
			// name= pname (prefix name) which is common|groupName|siteName + "-" + policyName
			name := pname + "-" + sFile.PolicyName
			err := CheckNameLength(namespace, name)
			if err != nil {
				panic(err)
			}

			sPolicyFile, err := ioutil.ReadFile(pbuilder.SourcePoliciesDir + "/" + sFile.FileName + utils.FileExt)
			if err != nil {
				panic(err)
			}
			_, resourceDef := pbuilder.getCustomResource(sFile.Data, sFile.Spec, sPolicyFile, rname, pbuilder.PolicyGenTemp.Metadata.Labels.Mcp)

			acmPolicy := pbuilder.getPolicy( name, namespace, resourceDef)
			policies[path + "/" + name] = acmPolicy
			subject := CreatePolicySubject(name)
			subjects = append(subjects, subject)
		}

		placementRule := CreatePlacementRule(pbuilder.PolicyGenTemp.Metadata.Name, namespace, matchKey, matchOper, matchValue)
		err := CheckNameLength(namespace, placementRule.Metadata.Name)
		if err != nil {
			panic(err)
		}
		policies[path + "/" + placementRule.Metadata.Name] = placementRule

		placementBinding := CreatePlacementBinding(pbuilder.PolicyGenTemp.Metadata.Name, namespace, placementRule.Metadata.Name, subjects)
		err = CheckNameLength(namespace, placementBinding.Metadata.Name)
		if err != nil {
			panic(err)
		}
		policies[path + "/" + placementBinding.Metadata.Name] = placementBinding
	} else if len(pbuilder.PolicyGenTemp.SourceFiles) != 0 && customResourseOnly {
		for id, sFile := range pbuilder.PolicyGenTemp.SourceFiles {
			_, rname := pbuilder.getPolicyName(id)
			sPolicyFile, err := ioutil.ReadFile(pbuilder.SourcePoliciesDir + "/" + sFile.FileName + utils.FileExt)
			if err != nil {
				panic(err)
			}
			rname, resourceDef := pbuilder.getCustomResource(sFile.Data, sFile.Spec, sPolicyFile, rname, pbuilder.PolicyGenTemp.Metadata.Labels.Mcp)
			policies[ utils.CustomResource + "/" + rname ] = resourceDef
		}
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

	policyObjDefArr := make([]utils.PolicyObjectDefinition, 1)
	policyObjDefArr[0] = policyObjDef

	acmPolicy := CreateAcmPolicy(name, namespace)
	err := CheckNameLength(namespace, name)
	if err != nil {
		panic(err)
	}
	acmPolicy.Spec.PolicyTemplates = policyObjDefArr
	return acmPolicy
}

func (pbuilder *PolicyBuilder) getCustomResource(data map[string]interface{},spec map[string]interface{}, sourcePolicy []byte, name string, mcp string) (string, map[string]interface{}) {
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
		sourcePolicyMap["spec"] = pbuilder.setValues(sourcePolicyMap["spec"].(map[string]interface{}), spec)
	}
	if len(data) != 0 {
		sourcePolicyMap["data"] = pbuilder.setValues(sourcePolicyMap["data"].(map[string]interface{}), data)
	}

	return name, sourcePolicyMap
}

func (pbuilder *PolicyBuilder) setValues(sourceMap map[string]interface{}, valueMap map[string]interface{}) map[string]interface{} {
	for k, v := range sourceMap {
		if valueMap[k] == nil {
			if reflect.ValueOf(v).Kind() == reflect.String && strings.HasPrefix(v.(string), "$") {
				delete(sourceMap, k)
			}
			continue
		}
		if reflect.ValueOf(sourceMap[k]).Kind() == reflect.Map {
			sourceMap[k] = pbuilder.setValues(v.(map[string]interface{}),valueMap[k].(map[string]interface{}))
		} else if reflect.ValueOf(v).Kind() == reflect.Slice ||
			reflect.ValueOf(v).Kind() == reflect.Array {
			intfArray := v.([]interface{})
			if len(intfArray) > 0 && reflect.ValueOf(intfArray[0]).Kind() == reflect.Map {
				tmpMapValues := make([]map[string]interface{}, len(intfArray))
				vIntfArray := valueMap[k].([]interface{})
				for id, intfMap := range intfArray {
					if id < len(vIntfArray) {
						tmpMapValues[id] = pbuilder.setValues(intfMap.(map[string]interface{}), vIntfArray[id].(map[string]interface{}))
					} else {
						tmpMapValues[id] = intfMap.(map[string]interface{})
					}
				}
				sourceMap[k] = tmpMapValues
			} else {
				sourceMap[k] = valueMap[k]
			}
		} else {
			sourceMap[k] = valueMap[k]
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
	if pbuilder.PolicyGenTemp.Metadata.Name != "" {
		if pbuilder.PolicyGenTemp.Metadata.Labels.SiteName != utils.NotApplicable {
			ns = utils.SiteNS
			matchKey = utils.Sites
			matchOper = utils.InOper
			matchValue = pbuilder.PolicyGenTemp.Metadata.Labels.SiteName
			path = utils.Sites + "/" + pbuilder.PolicyGenTemp.Metadata.Labels.SiteName
		} else if pbuilder.PolicyGenTemp.Metadata.Labels.GroupName != utils.NotApplicable {
			ns = utils.GroupNS
			matchKey = pbuilder.PolicyGenTemp.Metadata.Labels.GroupName
			matchOper = utils.ExistOper
			path = utils.Groups + "/" + pbuilder.PolicyGenTemp.Metadata.Labels.GroupName
		} else if pbuilder.PolicyGenTemp.Metadata.Labels.Common {
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
	if pbuilder.PolicyGenTemp.Metadata.Name != "" {
		if pbuilder.PolicyGenTemp.Metadata.Labels.SiteName != utils.NotApplicable {
			pname = pbuilder.PolicyGenTemp.Metadata.Labels.SiteName
		} else if pbuilder.PolicyGenTemp.Metadata.Labels.GroupName != utils.NotApplicable {
			pname = pbuilder.PolicyGenTemp.Metadata.Labels.GroupName
		} else if pbuilder.PolicyGenTemp.Metadata.Labels.Common {
			pname = utils.Common
		} else {
			panic("Error: missing metadata info either siteName, groupName or common should be set")
		}
		if len(pbuilder.PolicyGenTemp.SourceFiles) > sFileId {
			if pbuilder.PolicyGenTemp.SourceFiles[sFileId].Name != utils.NotApplicable &&
				pbuilder.PolicyGenTemp.SourceFiles[sFileId].Name != ""{
					rname = pbuilder.PolicyGenTemp.SourceFiles[sFileId].Name
			}
		}
	}
	// The names in the yaml must be compliant RFC 1123 domain names (all lower case)
	pname = strings.ToLower(pname)
	rname = strings.ToLower(rname)
	return pname, rname
}
