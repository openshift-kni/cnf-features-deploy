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

const McName = "predefined-extra-manifests"
const mcKind = "MachineConfig"

// merge the spec fields of all MC manifests except the ones that are in the doNotMerge list
func MergeManifests(individualMachineConfigs map[string]interface{}, doNotMerge map[string]bool) (map[string]interface{}, error) {
	// key is role, value is a list of MCs
	mergableMachineConfigs := make(map[string][]*machineconfigv1.MachineConfig)

	for filename, machineConfig := range individualMachineConfigs {
		if doNotMerge[filename] {
			continue
		}

		var data map[string]interface{}
		if err := yaml.Unmarshal([]byte(machineConfig.(string)), &data); err != nil {
			log.Printf("Error Could not unmarshal file content: (%s): %s\n", data, err)
			return individualMachineConfigs, err
		}
		// skip the manifest that is not a machine config
		if data["kind"] != mcKind {
			continue
		}

		err := addMachineConfig(data, mergableMachineConfigs)
		if err != nil {
			return individualMachineConfigs, err
		}
		// remove the individual file entries
		delete(individualMachineConfigs, filename)
	}

	for roleName, machineConfigs := range mergableMachineConfigs {
		var osImageURL string = ""
		//It only uses OSImageURL provided by the CVO
		merged, err := mcfgctrlcommon.MergeMachineConfigs(machineConfigs, osImageURL)
		if err != nil {
			return nil, err
		}

		merged.SetName(fmt.Sprintf("%s-%s", McName, roleName))
		merged.ObjectMeta.Labels = make(map[string]string)
		merged.ObjectMeta.Labels[machineconfigv1.MachineConfigRoleLabelKey] = roleName
		merged.TypeMeta.APIVersion = machineconfigv1.GroupVersion.String()
		merged.TypeMeta.Kind = mcKind

		// Marshal the machine config to json string
		b, err := json.Marshal(merged)
		if err != nil {
			log.Printf("Error: could not convert mc to json: (%s)\n", err)
			return nil, err
		}

		var m map[string]interface{}
		// Unmarshal the json string to interface for YamlTrim
		err = json.Unmarshal(b, &m)
		if err != nil {
			log.Printf("Error: could not convert json to map: (%s): %s\n", b, err)
			return nil, err
		}

		d := yamltrim.YamlTrim(m)
		if d == nil {
			return nil, fmt.Errorf("empty machineconfig")
		}
		// Marshal the interface to yaml bytes
		yamlBytes, err := yaml.Marshal(d)
		if err != nil {
			log.Printf("Error: could not convert map to yaml: (%s): %s\n", m, err)
			return nil, err
		}
		fileName := fmt.Sprintf("%s.yaml", merged.ObjectMeta.Name)
		individualMachineConfigs[fileName] = string(yamlBytes)
	}

	return individualMachineConfigs, nil
}

// convert yaml data to MC
func convertToMC(data map[string]interface{}) (*machineconfigv1.MachineConfig, error) {
	// Convert the yaml string to json
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error: could not convert map to json: (%s): %s\n", data, err)
		return nil, err
	}
	mc := machineconfigv1.MachineConfig{}
	// Convert the json string to machine config struct
	err = json.Unmarshal(jsonData, &mc)
	if err != nil {
		log.Printf("Error: could not convert json to mc: (%s): %s\n", jsonData, err)
		return nil, err
	}
	return &mc, nil
}

func addMachineConfig(data map[string]interface{}, configs map[string][]*machineconfigv1.MachineConfig) error {
	mc, err := convertToMC(data)
	if err != nil {
		return err
	}
	role := mc.ObjectMeta.Labels[machineconfigv1.MachineConfigRoleLabelKey]
	if configs[role] != nil {
		configs[role] = append(configs[role], mc)
	} else {
		configs[role] = []*machineconfigv1.MachineConfig{mc}
	}
	return nil
}
