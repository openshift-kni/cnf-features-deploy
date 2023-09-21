package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	fake "k8s.io/client-go/dynamic/fake"
	"sigs.k8s.io/yaml"
)

func clusterVersionProgressing() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "config.openshift.io/v1",
			"kind":       "ClusterVersion",
			"metadata": map[string]interface{}{
				"name": "version",
			},
			"spec": map[string]interface{}{},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":               "Progressing",
						"status":             "True",
						"lastTransitionTime": "2023-09-03T16:10:07Z",
					},
				},
			},
		},
	}
}

func clusterVersionNotProgressing() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "config.openshift.io/v1",
			"kind":       "ClusterVersion",
			"metadata": map[string]interface{}{
				"name": "version",
			},
			"spec": map[string]interface{}{},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":               "Progressing",
						"status":             "False",
						"lastTransitionTime": "2023-09-03T16:10:07Z",
					},
				},
			},
		},
	}
}

func clusterVersionNotFound() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "config.openshift.io/v1",
			"kind":       "ClusterVersion",
			"metadata": map[string]interface{}{
				"name": "version",
			},
			"spec": map[string]interface{}{},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":               "Faking",
						"status":             "False",
						"lastTransitionTime": "2023-09-03T16:10:07Z",
					},
				},
			},
		},
	}
}

func TestIsObjectStatusConditionPresentAndTrue(t *testing.T) {
	scheme := runtime.NewScheme()

	// Test found and positive
	client := fake.NewSimpleDynamicClient(scheme, clusterVersionProgressing())
	obj, err := client.Resource(clusterVersionResource()).Get(context.TODO(), "version", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	found, positive, err := isObjectStatusConditionPresentAndTrue(obj, "Progressing")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, found)
	assert.Equal(t, true, positive)

	// Test found and negative
	client = fake.NewSimpleDynamicClient(scheme, clusterVersionNotProgressing())
	obj, err = client.Resource(clusterVersionResource()).Get(context.TODO(), "version", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	found, positive, err = isObjectStatusConditionPresentAndTrue(obj, "Progressing")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, found)
	assert.Equal(t, false, positive)

	// Test not found
	client = fake.NewSimpleDynamicClient(scheme, clusterVersionNotFound())
	obj, err = client.Resource(clusterVersionResource()).Get(context.TODO(), "version", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	found, positive, err = isObjectStatusConditionPresentAndTrue(obj, "Progressing")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, false, found)
	assert.Equal(t, false, positive)
}

func TestCmToManifests(t *testing.T) {
	cmStr := `
 apiVersion: v1
 kind: ConfigMap
 metadata:
   name: ztp-post-provision
   namespace: ztp-profile
 data:
   sriov-nnp-cplane.yaml: |
     apiVersion: sriovnetwork.openshift.io/v1
     kind: SriovNetworkNodePolicy
     metadata:
       name: sriov-nnp-cplane
       namespace: openshift-sriov-network-operator
     spec:
       deviceType: netdevice
       isRdma: true
       nicSelector:
         pfNames:
         - ens2f0
       nodeSelector:
         node-role.kubernetes.io/worker: ""
       numVfs: 8
       priority: 10
       resourceName: cplane
   sriov-nnp-uplane.yaml: |
     apiVersion: sriovnetwork.openshift.io/v1
     kind: SriovNetworkNodePolicy
     metadata:
       name: sriov-nnp-uplane
       namespace: openshift-sriov-network-operator
     spec:
       deviceType: vfio-pci
       isRdma: false
       nicSelector:
         pfNames:
         - ens2f1
       nodeSelector:
         node-role.kubernetes.io/worker: ""
       numVfs: 8
       priority: 10
       resourceName: uplane
`
	var cm corev1.ConfigMap
	err := yaml.Unmarshal([]byte(cmStr), &cm)
	assert.Equal(t, nil, err)

	var manifests []unstructured.Unstructured
	err = cmToManifests(&cm, &manifests)
	assert.Equal(t, nil, err)
	assert.Equal(t, 2, len(manifests))

}
