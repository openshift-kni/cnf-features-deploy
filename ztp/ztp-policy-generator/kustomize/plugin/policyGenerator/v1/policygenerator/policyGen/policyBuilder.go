package policyGen

import (
	"bytes"
	utils "github.com/openshift-kni/cnf-features-deploy/ztp/ztp-policy-generator/kustomize/plugin/policyGenerator/v1/policygenerator/utils"
	yaml "gopkg.in/yaml.v3"
	"io"
	"reflect"
	"strconv"
	"strings"
)

type PolicyBuilder struct {
	PolicyGenTemp      utils.PolicyGenTemplate
	fHandler           *utils.FilesHandler
	customResourseOnly bool
}

func NewPolicyBuilder(PolicyGenTemp utils.PolicyGenTemplate, fileHandler *utils.FilesHandler, customResourseOnly bool) *PolicyBuilder {
	return &PolicyBuilder{PolicyGenTemp: PolicyGenTemp, fHandler: fileHandler, customResourseOnly: customResourseOnly}
}

func (pbuilder *PolicyBuilder) Build() map[string]interface{} {
	policies := make(map[string]interface{})

	if len(pbuilder.PolicyGenTemp.SourceFiles) != 0 && !pbuilder.customResourseOnly {
		if pbuilder.PolicyGenTemp.Metadata.Name == "" || pbuilder.PolicyGenTemp.Metadata.Name == utils.NotApplicable {
			panic("Error: missing policy template metadata.Name")
		}
		namespace, path, matchKey, matchValue, matchOper := pbuilder.getPolicyNsPath()
		subjects := make([]utils.Subject, 0)

		for _, sFile := range pbuilder.PolicyGenTemp.SourceFiles {
			pname := pbuilder.getPolicyName()
			// pname is the policyName prefix common|{groupName}|{siteName}
			name := pname + "-" + sFile.PolicyName
			if err := CheckNameLength(namespace, name); err != nil {
				panic(err)
			}

			sPolicyFile := pbuilder.fHandler.ReadSourceFileCR(sFile.FileName + utils.FileExt)
			resourcesDef := pbuilder.getCustomResources(sFile, sPolicyFile)
			acmPolicy := pbuilder.getPolicy(name, namespace, resourcesDef)
			policies[path+"/"+name] = acmPolicy

			subject := CreatePolicySubject(name)
			subjects = append(subjects, subject)
		}
		placementRule := CreatePlacementRule(pbuilder.PolicyGenTemp.Metadata.Name, namespace, matchKey, matchOper, matchValue)

		if err := CheckNameLength(namespace, placementRule.Metadata.Name); err != nil {
			panic(err)
		}
		policies[path+"/"+placementRule.Metadata.Name] = placementRule
		placementBinding := CreatePlacementBinding(pbuilder.PolicyGenTemp.Metadata.Name, namespace, placementRule.Metadata.Name, subjects)

		if err := CheckNameLength(namespace, placementBinding.Metadata.Name); err != nil {
			panic(err)
		}
		policies[path+"/"+placementBinding.Metadata.Name] = placementBinding
	} else if len(pbuilder.PolicyGenTemp.SourceFiles) != 0 && pbuilder.customResourseOnly {
		for _, sFile := range pbuilder.PolicyGenTemp.SourceFiles {
			sPolicyFile := pbuilder.fHandler.ReadSourceFileCR(sFile.FileName + utils.FileExt)
			resources := pbuilder.getCustomResources(sFile, sPolicyFile)

			for _, resource := range resources {
				name := resource["kind"].(string)
				name = name + "-" + resource["metadata"].(map[string]interface{})["name"].(string)

				if resource["metadata"].(map[string]interface{})["namespace"] != nil {
					name = name + "-" + resource["metadata"].(map[string]interface{})["namespace"].(string)
				}
				policies[utils.CustomResource+"/"+pbuilder.PolicyGenTemp.Metadata.Name+"/"+name] = resource
			}
		}
	}
	return policies
}

func (pbuilder *PolicyBuilder) getCustomResources(sFile utils.SourceFile, sPolicyFile []byte) []map[string]interface{} {
	yamls, err := pbuilder.splitYamls(sPolicyFile)
	resources := make([]map[string]interface{}, 0)

	if err != nil {
		panic(err)
	}
	// We are not allowing multiple yamls structure in same file to update its spec/data.
	if len(yamls) > 1 && (len(sFile.Data) > 0 || len(sFile.Spec) > 0) {
		panic("Update spec/data of multiple yamls structure in same file " + sFile.FileName +
			" not allowed. Instead separate them in multiple files")
	} else if len(yamls) > 1 && len(sFile.Data) == 0 && len(sFile.Spec) == 0 {
		for _, yaml := range yamls {
			resources = append(resources, pbuilder.getCustomResource(sFile, yaml, ""))
		}
	} else if len(yamls) == 1 {
		resources = append(resources, pbuilder.getCustomResource(sFile, yamls[0], pbuilder.PolicyGenTemp.Metadata.Labels.Mcp))
	}
	return resources
}

func (pbuilder *PolicyBuilder) getCustomResource(sourceFile utils.SourceFile, sourcePolicy []byte, mcp string) map[string]interface{} {
	sourcePolicyMap := make(map[string]interface{})
	sourcePolicyStr := string(sourcePolicy)

	if mcp != "" && mcp != utils.NotApplicable {
		sourcePolicyStr = strings.Replace(sourcePolicyStr, "$mcp", mcp, -1)
	}
	err := yaml.Unmarshal([]byte(sourcePolicyStr), &sourcePolicyMap)

	if err != nil {
		panic(err)
	}
	if sourceFile.Metadata.Name != "" && sourceFile.Metadata.Name != utils.NotApplicable {
		sourcePolicyMap["metadata"].(map[string]interface{})["name"] = sourceFile.Metadata.Name
	}
	if sourceFile.Metadata.Namespace != "" && sourceFile.Metadata.Namespace != utils.NotApplicable {
		sourcePolicyMap["metadata"].(map[string]interface{})["namespace"] = sourceFile.Metadata.Namespace
	}
	if len(sourceFile.Metadata.Labels) != 0 {
		sourcePolicyMap["metadata"].(map[string]interface{})["labels"] = sourceFile.Metadata.Labels
	}
	if len(sourceFile.Metadata.Annotations) != 0 {
		sourcePolicyMap["metadata"].(map[string]interface{})["annotations"] = sourceFile.Metadata.Annotations
	}
	if sourcePolicyMap["spec"] != nil {
		sourcePolicyMap["spec"] = pbuilder.setValues(sourcePolicyMap["spec"].(map[string]interface{}), sourceFile.Spec)
	}
	if sourcePolicyMap["data"] != nil {
		sourcePolicyMap["data"] = pbuilder.setValues(sourcePolicyMap["data"].(map[string]interface{}), sourceFile.Data)
	}
	return sourcePolicyMap
}

func (pbuilder *PolicyBuilder) setValues(sourceMap map[string]interface{}, valueMap map[string]interface{}) map[string]interface{} {
	for k, v := range sourceMap {
		if valueMap[k] == nil {
			if reflect.ValueOf(v).Kind() == reflect.String && (v.(string) == "" || strings.HasPrefix(v.(string), "$")) {
				delete(sourceMap, k)
			}
			continue
		}
		if reflect.ValueOf(sourceMap[k]).Kind() == reflect.Map {
			sourceMap[k] = pbuilder.setValues(v.(map[string]interface{}), valueMap[k].(map[string]interface{}))
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

func (pbuilder *PolicyBuilder) splitYamls(yamls []byte) ([][]byte, error) {
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

func (pbuilder *PolicyBuilder) getPolicy(name string, namespace string, resources []map[string]interface{}) utils.AcmPolicy {
	if err := CheckNameLength(namespace, name); err != nil {
		panic(err)
	}
	objTempArr := make([]utils.ObjectTemplates, 0)

	for _, resourse := range resources {
		objTempArr = append(objTempArr, CreateObjTemplates(resourse))
	}
	acmConfigPolicy := CreateAcmConfigPolicy(name, objTempArr)
	policyObjDef := CreatePolicyObjectDefinition(acmConfigPolicy)
	policyObjDefArr := make([]utils.PolicyObjectDefinition, 1)
	policyObjDefArr[0] = policyObjDef
	acmPolicy := CreateAcmPolicy(name, namespace, policyObjDefArr)

	return acmPolicy
}

func (pbuilder *PolicyBuilder) getPolicyNsPath() (string, string, string, string, string) {
	ns := ""
	path := ""
	matchKey := ""
	matchOper := ""
	matchValue := ""

	if pbuilder.PolicyGenTemp.Metadata.Name != "" {
		cval, err := strconv.ParseBool(pbuilder.PolicyGenTemp.Metadata.Labels.Common)
		if err != nil {
			cval = false
		}

		if pbuilder.PolicyGenTemp.Metadata.Labels.SiteName != utils.NotApplicable &&
			pbuilder.PolicyGenTemp.Metadata.Labels.SiteName != "" {
			ns = utils.SiteNS
			matchKey = utils.Sites
			matchOper = utils.InOper
			matchValue = pbuilder.PolicyGenTemp.Metadata.Labels.SiteName
			path = utils.Sites + "/" + pbuilder.PolicyGenTemp.Metadata.Labels.SiteName
		} else if pbuilder.PolicyGenTemp.Metadata.Labels.GroupName != utils.NotApplicable &&
			pbuilder.PolicyGenTemp.Metadata.Labels.GroupName != "" {
			ns = utils.GroupNS
			matchKey = pbuilder.PolicyGenTemp.Metadata.Labels.GroupName
			matchOper = utils.ExistOper
			path = utils.Groups + "/" + pbuilder.PolicyGenTemp.Metadata.Labels.GroupName
		} else if cval {
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

func (pbuilder *PolicyBuilder) getPolicyName() string {
	pname := ""
	cval, err := strconv.ParseBool(pbuilder.PolicyGenTemp.Metadata.Labels.Common)
	if err != nil {
		cval = false
	}

	if pbuilder.PolicyGenTemp.Metadata.Labels.SiteName != utils.NotApplicable &&
		pbuilder.PolicyGenTemp.Metadata.Labels.SiteName != "" {
		pname = pbuilder.PolicyGenTemp.Metadata.Labels.SiteName
	} else if pbuilder.PolicyGenTemp.Metadata.Labels.GroupName != utils.NotApplicable &&
		pbuilder.PolicyGenTemp.Metadata.Labels.GroupName != "" {
		pname = pbuilder.PolicyGenTemp.Metadata.Labels.GroupName
	} else if cval {
		pname = utils.Common
	} else {
		panic("Error: missing metadata info either siteName, groupName or common should be set")
	}
	// The names in the yaml must be compliant RFC 1123 domain names (all lower case)
	return strings.ToLower(pname)
}
