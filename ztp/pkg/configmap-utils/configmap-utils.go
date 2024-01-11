package cmutils

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func WrapObjects(objects []unstructured.Unstructured, name string, namespace string) (*v1.ConfigMap, error) {
	cm := v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{},
	}

	for _, item := range objects {
		key := fmt.Sprintf("%s.yaml", item.GetName())
		out, err := yaml.Marshal(item.Object)
		if err != nil {
			return &cm, err
		}
		cm.Data[key] = string(out)
	}
	return &cm, nil
}
