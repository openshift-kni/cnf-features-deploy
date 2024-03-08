package placement

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/test-network-function/pgt2acm/packages/fileutils"
	"gopkg.in/yaml.v2"
)

type Placement struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Spec struct {
		Predicates  []Predicate  `yaml:"predicates"`
		Tolerations []Toleration `yaml:"tolerations"`
	} `yaml:"spec"`
}
type Predicate struct {
	RequiredClusterSelector struct {
		LabelSelector map[string]interface{} `yaml:"labelSelector"`
	} `yaml:"requiredClusterSelector"`
}
type Toleration struct {
	Key      string `yaml:"key"`
	Operator string `yaml:"operator"`
}

func GeneratePlacementFile(policyName, policyNamespace, outputDir string, labelSelector map[string]interface{}) (placementPathRelative string, err error) {
	placement := Placement{APIVersion: "cluster.open-cluster-management.io/v1beta1",
		Kind: "Placement"}
	placement.Metadata.Name = "placement-" + policyName
	placement.Metadata.Namespace = policyNamespace

	// adding predicate
	predicate := Predicate{}
	predicate.RequiredClusterSelector.LabelSelector = labelSelector
	placement.Spec.Predicates = append(placement.Spec.Predicates, predicate)

	// toleration to create child policies even if the managed cluster is not yet available
	toleration := Toleration{}
	toleration.Key = "cluster.open-cluster-management.io/unreachable"
	toleration.Operator = "Exists"
	placement.Spec.Tolerations = append(placement.Spec.Tolerations, toleration)

	placementPathRelative = policyName + "-placement.yaml"
	placementPath := filepath.Join(outputDir, placementPathRelative)

	err = writePlacementToFile(&placement, placementPath)
	if err != nil {
		return placementPathRelative, fmt.Errorf("error writing placement to file, err: %s", err)
	}
	return placementPathRelative, nil
}

func writePlacementToFile(placement *Placement, outputFile string) (err error) {
	contentYAML, err := yaml.Marshal(placement)
	if err != nil {
		return fmt.Errorf("could not marshall placement, err: %s", err)
	}

	contentYAML = []byte("---\n" + string(contentYAML))

	// Ensure the directory exists
	dir := filepath.Dir(outputFile)
	err = os.MkdirAll(dir, fileutils.DefaultDirWritePermissions)
	if err != nil {
		return err
	}

	err = os.WriteFile(outputFile, contentYAML, fileutils.DefaultFileWritePermissions)
	if err != nil {
		fmt.Printf("Error writing to file, err: %s", err)
		return err
	}
	fmt.Printf("Wrote placement file: %s\n", outputFile)
	return nil
}
