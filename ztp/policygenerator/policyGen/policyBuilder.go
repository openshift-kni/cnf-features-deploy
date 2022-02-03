package policyGen

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	utils "github.com/openshift-kni/cnf-features-deploy/ztp/policygenerator/utils"
	yaml "gopkg.in/yaml.v3"
)

type PolicyBuilder struct {
	fHandler *utils.FilesHandler
}

// struct used to keep the user PGT sourceFile data with the actual built CR
// from that source file.
type generatedCR struct {
	globalComplianceType string
	pgtSourceFile        utils.SourceFile
	builtCR              map[string]interface{}
}

func NewPolicyBuilder(fileHandler *utils.FilesHandler) *PolicyBuilder {
	return &PolicyBuilder{fHandler: fileHandler}
}

func (pbuilder *PolicyBuilder) Build(policyGenTemp utils.PolicyGenTemplate) (map[string]interface{}, error) {
	policies := make(map[string]interface{})

	if policyGenTemp.Metadata.Name == "" || policyGenTemp.Metadata.Namespace == "" {
		return policies, errors.New("PolicyGenTemplate Metadata.Name & Metadata.Namespace must be defined")
	}

	if len(policyGenTemp.Spec.SourceFiles) > 0 {
		subjects := make([]utils.Subject, 0)
		for _, sFile := range policyGenTemp.Spec.SourceFiles {
			sPolicyFile, err := pbuilder.fHandler.ReadSourceFile(sFile.FileName)
			if err != nil {
				return policies, err
			}

			resources, err := pbuilder.getCustomResources(sFile, sPolicyFile, policyGenTemp.Spec.Mcp)
			if err != nil {
				return policies, err
			}
			if sFile.PolicyName == "" || !policyGenTemp.Spec.WrapInPolicy {
				// Generate plain CRs (no policy)
				var name string
				if len(resources) > 1 {
					// Multi-document yaml - use the source filename
					name = fmt.Sprintf("Multiple-%s", strings.TrimSuffix(sFile.FileName, filepath.Ext(sFile.FileName)))
				} else {
					// Single-resource yaml - construct a filename based on the contents
					resource := resources[0]
					nameParts := []string{
						resource["kind"].(string),
						resource["metadata"].(map[string]interface{})["name"].(string),
					}
					resourceNamespace := resource["metadata"].(map[string]interface{})["namespace"]
					if resourceNamespace != nil {
						nameParts = append(nameParts, resourceNamespace.(string))
					}
					name = strings.Join(nameParts, "-")
				}
				output := path.Join(utils.CustomResource, policyGenTemp.Metadata.Name, name)
				policies[output] = resources
			} else {
				// Generate a policy-wrapped CR, with the filename based on the policy and source filename
				name := strings.Join([]string{policyGenTemp.Metadata.Name, sFile.PolicyName}, "-")
				output := path.Join(policyGenTemp.Metadata.Name, name)
				var acmPolicy utils.AcmPolicy
				// resources is a map of only the built CRs. Bind them with sourceFile
				// data as needed by low level methods setting various policy attributes
				annotatedResources := make([]generatedCR, len(resources))
				for idx, cr := range resources {
					annotatedResources[idx] = generatedCR{
						globalComplianceType: policyGenTemp.Spec.ComplianceType,
						pgtSourceFile:        sFile,
						builtCR:              cr,
					}
				}
				if sFile.PolicyName != "" && policies[output] == nil {
					// Generate new policy
					acmPolicy, err = pbuilder.createAcmPolicy(name, policyGenTemp.Metadata.Namespace, annotatedResources)
					if err != nil {
						return policies, err
					}
					//set to default remediationAction or user set from PGT
					remediationActionVal := policyGenTemp.Spec.RemediationAction
					if sFile.RemediationAction != utils.UnsetStringValue {
						// sourceFile remediationAction overrides the default remediationAction
						remediationActionVal = sFile.RemediationAction
					}

					acmPolicy.Spec.RemediationAction = remediationActionVal
					acmPolicy.Spec.PolicyTemplates[0].ObjDef.Spec.RemediationAction = remediationActionVal

					subject := CreatePolicySubject(name)
					subjects = append(subjects, subject)
				} else if sFile.PolicyName != "" && policies[output] != nil {
					// Append new CR to policy
					acmPolicy, err = pbuilder.AppendAcmPolicy(policies[output].(utils.AcmPolicy), annotatedResources)
					if err != nil {
						return policies, err
					}
					if sFile.RemediationAction != utils.UnsetStringValue {

						if sFile.RemediationAction != policies[output].(utils.AcmPolicy).Spec.RemediationAction &&
							policies[output].(utils.AcmPolicy).Spec.RemediationAction != policyGenTemp.Spec.RemediationAction {
							//remediationAction already overridden by a different sourceFile
							// and the remediationAction does not match
							return policies, errors.New("remediationAction conflict for policyName " + sFile.PolicyName)
						} else {
							// sourceFile level remediationAction overrides default/PGT level
							acmPolicy.Spec.RemediationAction = sFile.RemediationAction
							acmPolicy.Spec.PolicyTemplates[0].ObjDef.Spec.RemediationAction = sFile.RemediationAction
						}
					}
				}
				policies[output] = acmPolicy
			}
		}
		if len(subjects) > 0 {
			// Create rules
			if err := CheckBindingRules(policyGenTemp.Metadata.Name,
				policyGenTemp.Spec.BindingRules, policyGenTemp.Spec.BindingExcludedRules); err != nil {
				return policies, err
			}
			placementRule := CreatePlacementRule(policyGenTemp.Metadata.Name, policyGenTemp.Metadata.Namespace,
				policyGenTemp.Spec.BindingRules, policyGenTemp.Spec.BindingExcludedRules)

			if err := CheckNameLength(placementRule.Metadata.Namespace, placementRule.Metadata.Name); err != nil {
				return policies, err
			}
			policies[policyGenTemp.Metadata.Name+"/"+placementRule.Metadata.Name] = placementRule

			// Create binding
			placementBinding := CreatePlacementBinding(policyGenTemp.Metadata.Name, policyGenTemp.Metadata.Namespace, placementRule.Metadata.Name, subjects)

			if err := CheckNameLength(placementBinding.Metadata.Namespace, placementBinding.Metadata.Name); err != nil {
				return policies, err
			}
			policies[policyGenTemp.Metadata.Name+"/"+placementBinding.Metadata.Name] = placementBinding
		}
	}

	return policies, nil
}

func (pbuilder *PolicyBuilder) getCustomResources(sFile utils.SourceFile, sourceCRFile []byte, mcp string) ([]map[string]interface{}, error) {
	yamls, err := pbuilder.splitYamls(sourceCRFile)
	resources := make([]map[string]interface{}, 0)
	if err != nil {
		return resources, err
	}
	// Update multiple yamls structure in same file not allowed.
	if len(yamls) > 1 && (len(sFile.Data) > 0 || len(sFile.Spec) > 0) {
		return resources, errors.New("Update spec/data of multiple yamls structure in same file " + sFile.FileName +
			" not allowed. Instead separate them in multiple files")
	} else if len(yamls) > 1 && len(sFile.Data) == 0 && len(sFile.Spec) == 0 {
		// Append yaml structures without modify spec or data fields
		for _, yaml := range yamls {
			resource, err := pbuilder.getCustomResource(sFile, yaml, mcp)
			if err != nil {
				return resources, err
			}
			resources = append(resources, resource)
		}
	} else if len(yamls) == 1 {
		resource, err := pbuilder.getCustomResource(sFile, yamls[0], mcp)
		if err != nil {
			return resources, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (pbuilder *PolicyBuilder) getCustomResource(sourceFile utils.SourceFile, sourceCR []byte, mcp string) (map[string]interface{}, error) {
	resourceMap := make(map[string]interface{})
	resourceStr := string(sourceCR)

	if mcp != "" {
		resourceStr = strings.Replace(resourceStr, "$mcp", mcp, -1)
	}
	err := yaml.Unmarshal([]byte(resourceStr), &resourceMap)

	if err != nil {
		return resourceMap, err
	}
	if sourceFile.Metadata.Name != "" {
		resourceMap["metadata"].(map[string]interface{})["name"] = sourceFile.Metadata.Name
	}
	if sourceFile.Metadata.Namespace != "" {
		resourceMap["metadata"].(map[string]interface{})["namespace"] = sourceFile.Metadata.Namespace
	}
	if len(sourceFile.Metadata.Labels) != 0 {
		// convert from map[string]string to map[string]interface{}
		sourceFileLabels := make(map[string]interface{})
		for k, v := range sourceFile.Metadata.Labels {
			sourceFileLabels[k] = v
		}

		if resourceMap["metadata"].(map[string]interface{})["labels"] == nil {
			// source CR does not have labels
			resourceMap["metadata"].(map[string]interface{})["labels"] = make(map[string]interface{})
		}
		resourceMap["metadata"].(map[string]interface{})["labels"] = pbuilder.setValues(
			resourceMap["metadata"].(map[string]interface{})["labels"].(map[string]interface{}), sourceFileLabels)
	}
	if len(sourceFile.Metadata.Annotations) != 0 {
		// convert from map[string]string to map[string]interface{}
		sourceFileAnnotations := make(map[string]interface{})
		for k, v := range sourceFile.Metadata.Annotations {
			sourceFileAnnotations[k] = v
		}

		if resourceMap["metadata"].(map[string]interface{})["annotations"] == nil {
			// source CR does not have annotations
			resourceMap["metadata"].(map[string]interface{})["annotations"] = make(map[string]interface{})
		}
		resourceMap["metadata"].(map[string]interface{})["annotations"] = pbuilder.setValues(
			resourceMap["metadata"].(map[string]interface{})["annotations"].(map[string]interface{}), sourceFileAnnotations)
	}

	if resourceMap["spec"] != nil {
		resourceMap["spec"] = pbuilder.setValues(resourceMap["spec"].(map[string]interface{}), sourceFile.Spec)
	} else if sourceFile.Spec != nil {
		// If the user supplies a "spec" section but the source CR does not have
		// one, this will ensure we pull in the user content
		resourceMap["spec"] = make(map[string]interface{})
		resourceMap["spec"] = pbuilder.setValues(resourceMap["spec"].(map[string]interface{}), sourceFile.Spec)
	}
	if resourceMap["data"] != nil {
		resourceMap["data"] = pbuilder.setValues(resourceMap["data"].(map[string]interface{}), sourceFile.Data)
	} else if sourceFile.Data != nil {
		// If the user supplies a "data" section but the source CR does not have
		// one, this will ensure we pull in the user content
		resourceMap["data"] = make(map[string]interface{})
		resourceMap["data"] = pbuilder.setValues(resourceMap["data"].(map[string]interface{}), sourceFile.Data)
	}

	return resourceMap, nil
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

	// If the user provides values in the PGT spec/data sections for which there
	// is not a corresponding key in the source-cr we need to simply copy over
	// the user supplied values
	for k, v := range valueMap {
		// Merging user content into fields that are already in the source-cr is
		// handled in previous loop
		if sourceMap[k] == nil {
			sourceMap[k] = v
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

func (pbuilder *PolicyBuilder) createAcmPolicy(name string, namespace string, resources []generatedCR) (utils.AcmPolicy, error) {
	if err := CheckNameLength(namespace, name); err != nil {
		return utils.AcmPolicy{}, err
	}

	acmPolicy := utils.AcmPolicy{}
	err := yaml.Unmarshal([]byte(acmPolicyTemplate), &acmPolicy)
	if err != nil {
		return acmPolicy, err
	}
	acmPolicy.Metadata.Name = name
	acmPolicy.Metadata.Namespace = namespace

	if len(acmPolicy.Spec.PolicyTemplates) < 1 {
		return acmPolicy, errors.New("Mising Policy template in the " + acmPolicyTemplate)
	}
	acmPolicy.Spec.PolicyTemplates[0].ObjDef.Metadata.Name = name + "-config"

	if len(acmPolicy.Spec.PolicyTemplates[0].ObjDef.Spec.ObjectTemplates) < 1 {
		return acmPolicy, errors.New("Mising Object template in the " + acmPolicyTemplate)
	}
	objTempArr := make([]utils.ObjectTemplates, len(resources))

	for idx, resource := range resources {
		objTemp := BuildObjectTemplate(resource)
		objTempArr[idx] = objTemp

		if err = SetPolicyDeployWave(acmPolicy.Metadata, resource); err != nil {
			return acmPolicy, err
		}
	}
	acmPolicy.Spec.PolicyTemplates[0].ObjDef.Spec.ObjectTemplates = objTempArr
	return acmPolicy, nil
}

func (pbuilder *PolicyBuilder) AppendAcmPolicy(acmPolicy utils.AcmPolicy, resources []generatedCR) (utils.AcmPolicy, error) {
	if len(acmPolicy.Spec.PolicyTemplates) < 1 {
		return acmPolicy, errors.New("Mising Policy template in the " + acmPolicy.Metadata.Name)
	}

	if len(acmPolicy.Spec.PolicyTemplates[0].ObjDef.Spec.ObjectTemplates) < 1 {
		return acmPolicy, errors.New("Mising Object template in the " + acmPolicy.Metadata.Name)
	}
	objTempArr := acmPolicy.Spec.PolicyTemplates[0].ObjDef.Spec.ObjectTemplates

	for _, resource := range resources {
		objTemp := BuildObjectTemplate(resource)
		objTempArr = append(objTempArr, objTemp)

		if err := SetPolicyDeployWave(acmPolicy.Metadata, resource); err != nil {
			return acmPolicy, err
		}
	}
	acmPolicy.Spec.PolicyTemplates[0].ObjDef.Spec.ObjectTemplates = objTempArr

	return acmPolicy, nil
}
