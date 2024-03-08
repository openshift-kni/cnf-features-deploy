package labels

import (
	"encoding/json"
	"fmt"

	yamlconv "github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

// LabelToSelector Converts a label list and an exclude label list to a Selector
func LabelToSelector(labelList, excludeLabelList map[string]string) (selector labels.Selector) {
	selector = labels.NewSelector()
	selector = addRequirements(labelList, selector, false)
	selector = addRequirements(excludeLabelList, selector, true)
	return selector
}

// addRequirements Converts a label list to a list of requirements in a selector
func addRequirements(labelList map[string]string, selector labels.Selector, exclude bool) labels.Selector {
	for key, value := range labelList {
		if key == "" {
			continue
		}
		if exclude {
			requirements, _ := labels.NewRequirement(key, selection.DoesNotExist, []string{value})
			selector = selector.Add(*requirements)
			continue
		}
		if value == "" {
			requirements, _ := labels.NewRequirement(key, selection.Exists, []string{})
			selector = selector.Add(*requirements)
			continue
		}
		if value != "" {
			requirements, _ := labels.NewRequirement(key, selection.In, []string{value})
			selector = selector.Add(*requirements)
			continue
		}
	}
	return selector
}

// OutputGeneric Outputs the Selector as a generic Yaml
func OutputGeneric(selector labels.Selector) (output map[string]interface{}, err error) {
	labelSelector, err := metav1.ParseToLabelSelector(selector.String())
	if err != nil {
		return output, err
	}
	jsonText, err := json.Marshal(labelSelector)
	if err != nil {
		return output, fmt.Errorf("failed to unMarshall label selector to json: %v, err: %s", selector, err)
	}
	var yamlText []byte
	yamlText, err = yamlconv.JSONToYAML(jsonText)
	if err != nil {
		return output, fmt.Errorf("failed to convert label selector to yaml: %v, err: %s", selector, err)
	}
	err = yaml.Unmarshal(yamlText, &output)
	if err != nil {
		return output, fmt.Errorf("failed to Marshall label selector: %v, err: %s", selector, err)
	}
	return output, nil
}
