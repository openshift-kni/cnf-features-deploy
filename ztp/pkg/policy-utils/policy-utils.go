package policyutils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	policyv1 "open-cluster-management.io/config-policy-controller/api/v1"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
)

type PolicyExtractor struct {
	PolicyInterface func(schema.GroupVersionResource) dynamic.NamespaceableResourceInterface
	Ctx             context.Context
	Options         metav1.ListOptions
}

func PolicyResourceSchema() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "policy.open-cluster-management.io",
		Version:  "v1",
		Resource: "policies",
	}
}

// Gets typed policies from the specified namespace
func (e *PolicyExtractor) GetPoliciesForNamespace(namespace string) ([]policiesv1.Policy, error) {
	if e.PolicyInterface == nil || e.Ctx == nil {
		return nil, fmt.Errorf("uninitialized PolicyExtractor")
	}

	var extracted []policiesv1.Policy

	pl, err := e.PolicyInterface(PolicyResourceSchema()).Namespace(namespace).List(e.Ctx, e.Options)
	if err != nil {
		return extracted, err
	}
	for _, item := range pl.Items {
		var pol policiesv1.Policy
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &pol)
		if err != nil {
			return extracted, err
		}
		extracted = append(extracted, pol)
	}
	return extracted, nil
}

// Gets encapsulated objects from configuration policies
func GetConfigurationObjects(policies []policiesv1.Policy) ([]unstructured.Unstructured, error) {
	var uobjects []unstructured.Unstructured

	var objects []runtime.RawExtension
	for _, pol := range policies {
		if !pol.Spec.Disabled {
			for _, template := range pol.Spec.PolicyTemplates {
				o := *template.ObjectDefinition.DeepCopy()
				objects = append(objects, o)
			}
		}
	}

	for _, ob := range objects {
		var pol policyv1.ConfigurationPolicy
		err := json.Unmarshal(ob.DeepCopy().Raw, &pol)
		if err != nil {
			log.Print(err)
			return uobjects, err
		}
		for _, ot := range pol.Spec.ObjectTemplates {
			var object unstructured.Unstructured
			err = object.UnmarshalJSON(ot.ObjectDefinition.DeepCopy().Raw)
			if err != nil {
				return uobjects, err
			}
			object.Object["status"] = map[string]interface{}{} // remove status, we can't apply it
			uobjects = append(uobjects, object)
		}
	}
	return uobjects, nil
}
