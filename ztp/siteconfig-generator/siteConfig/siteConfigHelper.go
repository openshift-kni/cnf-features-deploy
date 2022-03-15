package siteConfig

import (
	"encoding/json"
	"fmt"
	"log"

	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcfgctrlcommon "github.com/openshift/machine-config-operator/pkg/controller/common"

	"github.com/lack/yamltrim"
	yaml "gopkg.in/yaml.v3"
)

const mcRoleKey = "machineconfiguration.openshift.io/role"
const mcName = "predefined-extra-manifests"

func isExcluded(s []string, str string) bool {
	if s == nil {
		return false
	}
	for _, v := range s {
		if str == v {
			return true
		}
	}
	return false
}

// merge the spec fields of all MC manifests except the ones that are in the excluded list
func MergeManifests(dataMap map[string]interface{}, excludes []string) (map[string]interface{}, error) {
	// key is role, value is a list of MCs
	configs := make(map[string][]*machineconfigv1.MachineConfig)

	for k, v := range dataMap {
		if isExcluded(excludes, k) {
			continue
		}
		err := addMachineConfig(v, configs)
		if err != nil {
			return dataMap, err
		}
		// remove the individual file entries
		delete(dataMap, k)
	}

	for k, v := range configs {
		var osImageURL string = ""
		//It only uses OSImageURL provided by the CVO
		merged, err := mcfgctrlcommon.MergeMachineConfigs(v, osImageURL)
		if err != nil {
			return nil, err
		}

		merged.SetName(fmt.Sprintf("%s-%s", mcName, k))
		merged.ObjectMeta.Labels = make(map[string]string)
		merged.ObjectMeta.Labels[mcRoleKey] = k
		merged.TypeMeta.APIVersion = machineconfigv1.GroupVersion.String()
		merged.TypeMeta.Kind = "MachineConfig"

		// Marshal to json
		b, err := json.Marshal(merged)
		if err != nil {
			log.Printf("Error: could not convert mc to json: (%s)\n", err)
			return nil, err
		}

		var m map[string]interface{}
		err = json.Unmarshal(b, &m)
		if err != nil {
			log.Printf("Error: could not convert json to map: (%s): %s\n", b, err)
			return nil, err
		}

		d := yamltrim.YamlTrim(m)
		if d == nil {
			return nil, fmt.Errorf("empty machineconfig")
		}

		yamlBytes, err := yaml.Marshal(d)
		if err != nil {
			log.Printf("Error: could not convert map to yaml: (%s): %s\n", m, err)
			return nil, err
		}
		fileName := fmt.Sprintf("%s.yaml", merged.ObjectMeta.Name)
		dataMap[fileName] = string(yamlBytes)
	}

	return dataMap, nil
}

// convert yaml data to MC
func convertToMC(data interface{}) (*machineconfigv1.MachineConfig, error) {
	var m map[string]interface{}

	err := yaml.Unmarshal([]byte(data.(string)), &m)
	if err != nil {
		log.Printf("Error Could not unmarshal file content: (%s): %s\n", data, err)
		return nil, err
	}

	jsonData, err := json.Marshal(m)
	if err != nil {
		log.Printf("Error: could not convert map to json: (%s): %s\n", m, err)
		return nil, err
	}
	mc := machineconfigv1.MachineConfig{}
	err = json.Unmarshal(jsonData, &mc)
	if err != nil {
		log.Printf("Error: could not convert json to mc: (%s): %s\n", jsonData, err)
		return nil, err
	}
	return &mc, nil
}

func addMachineConfig(data interface{}, configs map[string][]*machineconfigv1.MachineConfig) error {
	mc, err := convertToMC(data)
	if err != nil {
		return err
	}
	role := mc.ObjectMeta.Labels[mcRoleKey]
	if _, ok := configs[role]; ok {
		configs[role] = append(configs[role], mc)
	} else {
		var mcPtr []*machineconfigv1.MachineConfig
		configs[role] = append(mcPtr, mc)
	}
	return nil
}
