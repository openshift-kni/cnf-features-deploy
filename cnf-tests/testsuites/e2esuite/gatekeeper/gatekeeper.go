package gatekeeper

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testClient "github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/pods"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"

	gkv1alpha "github.com/open-policy-agent/gatekeeper/apis/mutations/v1alpha1"
)

const (
	// TestingNamespace is the namespace for resources in this test
	TestingNamespace = "gatekeeper-testing"
)

var _ = Describe("gatekeeper", func() {
	client := testClient.Client

	AfterEach(func() {
		err := deletePods(TestingNamespace, client)
		Expect(err).NotTo(HaveOccurred())

		namespacesUsed := []string{"mutation-included", "mutation-excluded", "gk-test-object"}

		for _, namespace := range namespacesUsed {
			if namespaces.Exists(namespace, client) {
				err := client.Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
		}

		for _, namespace := range namespacesUsed {
			err := namespaces.WaitForDeletion(client, namespace, time.Minute)
			Expect(err).ToNot(HaveOccurred())
		}

		amList := &gkv1alpha.AssignMetadataList{}

		err = client.List(context.Background(), amList)
		Expect(err).NotTo(HaveOccurred())

		for _, am := range amList.Items {
			err := client.Delete(context.Background(), &am)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("should be able to add metadata info(labels/annotations)",
		func() {
			amList := []*gkv1alpha.AssignMetadata{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "add-label",
					},
					Spec: gkv1alpha.AssignMetadataSpec{
						Match: gkv1alpha.Match{
							Scope:      apiextensionsv1beta1.NamespaceScoped,
							Namespaces: []string{TestingNamespace},
							Kinds: []gkv1alpha.Kinds{
								{
									APIGroups: []string{""},
									Kinds:     []string{"Pod"},
								},
							},
						},
						Location: "metadata.labels.mutated",
						Parameters: gkv1alpha.MetadataParameters{
							Assign: runtime.RawExtension{
								Raw: []byte(fmt.Sprintf("{\"value\":\"true\"}")),
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "add-annotation",
					},
					Spec: gkv1alpha.AssignMetadataSpec{
						Match: gkv1alpha.Match{
							Scope:      apiextensionsv1beta1.NamespaceScoped,
							Namespaces: []string{TestingNamespace},
							Kinds: []gkv1alpha.Kinds{
								{
									APIGroups: []string{""},
									Kinds:     []string{"Pod"},
								},
							},
						},
						Location: "metadata.annotations.mutated",
						Parameters: gkv1alpha.MetadataParameters{
							Assign: runtime.RawExtension{
								Raw: []byte(fmt.Sprintf("{\"value\":\"true\"}")),
							},
						},
					},
				},
			}

			for _, am := range amList {
				err := client.Create(context.Background(), am)
				Expect(err).NotTo(HaveOccurred())
			}

			pod := pods.DefinePod(TestingNamespace)
			err := client.Create(context.Background(), pod)
			Expect(err).NotTo(HaveOccurred())

			podKey, err := k8sClient.ObjectKeyFromObject(pod)
			Expect(err).ToNot(HaveOccurred())
			err = client.Get(context.Background(), podKey, pod)
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.GetLabels()["mutated"]).To(Equal("true"))
			Expect(pod.GetAnnotations()["mutated"]).To(Equal("true"))
		},
	)

	It("should avoid mutating existing metadata info(labels/annotations)",
		func() {
			amList := []*gkv1alpha.AssignMetadata{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mutate-label",
					},
					Spec: gkv1alpha.AssignMetadataSpec{
						Match: gkv1alpha.Match{
							Scope:      apiextensionsv1beta1.NamespaceScoped,
							Namespaces: []string{TestingNamespace},
							Kinds: []gkv1alpha.Kinds{
								{
									APIGroups: []string{""},
									Kinds:     []string{"Pod"},
								},
							},
						},
						Location: "metadata.labels.mutated",
						Parameters: gkv1alpha.MetadataParameters{
							Assign: runtime.RawExtension{
								Raw: []byte(fmt.Sprintf("{\"value\":\"true\"}")),
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mutate-annotation",
					},
					Spec: gkv1alpha.AssignMetadataSpec{
						Match: gkv1alpha.Match{
							Scope:      apiextensionsv1beta1.NamespaceScoped,
							Namespaces: []string{TestingNamespace},
							Kinds: []gkv1alpha.Kinds{
								{
									APIGroups: []string{""},
									Kinds:     []string{"Pod"},
								},
							},
						},
						Location: "metadata.annotations.mutated",
						Parameters: gkv1alpha.MetadataParameters{
							Assign: runtime.RawExtension{
								Raw: []byte(fmt.Sprintf("{\"value\":\"true\"}")),
							},
						},
					},
				},
			}

			for _, am := range amList {
				err := client.Create(context.Background(), am)
				Expect(err).NotTo(HaveOccurred())
			}

			pod := pods.DefinePod(TestingNamespace)
			labels := map[string]string{
				"mutated": "false",
			}
			annotations := map[string]string{
				"mutated": "false",
			}
			pod.SetLabels(labels)
			pod.SetAnnotations(annotations)
			err := client.Create(context.Background(), pod)
			Expect(err).NotTo(HaveOccurred())

			podKey, err := k8sClient.ObjectKeyFromObject(pod)
			Expect(err).ToNot(HaveOccurred())
			err = client.Get(context.Background(), podKey, pod)
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.GetLabels()["mutated"]).To(Equal("false"))
			Expect(pod.GetAnnotations()["mutated"]).To(Equal("false"))
		},
	)

	It("should apply mutations by order", func() {
		By("Creating assignMetadata mutation-b")
		amB := &gkv1alpha.AssignMetadata{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mutation-b",
			},
			Spec: gkv1alpha.AssignMetadataSpec{
				Match: gkv1alpha.Match{
					Scope:      apiextensionsv1beta1.NamespaceScoped,
					Namespaces: []string{TestingNamespace},
					Kinds: []gkv1alpha.Kinds{
						{
							APIGroups: []string{""},
							Kinds:     []string{"Pod"},
						},
					},
				},
				Location: "metadata.labels.mutated-by",
				Parameters: gkv1alpha.MetadataParameters{
					Assign: runtime.RawExtension{
						Raw: []byte(fmt.Sprintf("{\"value\":\"b\"}")),
					},
				},
			},
		}
		err := client.Create(context.Background(), amB)
		Expect(err).NotTo(HaveOccurred())

		By("Creating test-pod-b")
		testPodB := pods.DefinePod(TestingNamespace)
		testPodB.SetName("test-pod-b")
		err = client.Create(context.Background(), testPodB)
		Expect(err).NotTo(HaveOccurred())

		By("Asserting that test-pod-b was labeled")
		testPodBKey, err := k8sClient.ObjectKeyFromObject(testPodB)
		Expect(err).NotTo(HaveOccurred())
		err = client.Get(context.Background(), testPodBKey, testPodB)
		Expect(err).NotTo(HaveOccurred())
		Expect(testPodB.GetLabels()["mutated-by"]).To(Equal("b"))

		By("Creating assignMetadata mutation-a")
		amA := &gkv1alpha.AssignMetadata{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mutation-a",
			},
			Spec: gkv1alpha.AssignMetadataSpec{
				Match: gkv1alpha.Match{
					Scope:      apiextensionsv1beta1.NamespaceScoped,
					Namespaces: []string{TestingNamespace},
					Kinds: []gkv1alpha.Kinds{
						{
							APIGroups: []string{""},
							Kinds:     []string{"Pod"},
						},
					},
				},
				Location: "metadata.labels.mutated-by",
				Parameters: gkv1alpha.MetadataParameters{
					Assign: runtime.RawExtension{
						Raw: []byte(fmt.Sprintf("{\"value\":\"a\"}")),
					},
				},
			},
		}
		err = client.Create(context.Background(), amA)
		Expect(err).NotTo(HaveOccurred())

		By("Creating test-pod-a")
		testPodA := pods.DefinePod(TestingNamespace)
		testPodA.SetName("test-pod-a")
		err = client.Create(context.Background(), testPodA)
		Expect(err).NotTo(HaveOccurred())

		By("Asserting that test-pod-a was labeled")
		testPodAKey, err := k8sClient.ObjectKeyFromObject(testPodA)
		Expect(err).ToNot(HaveOccurred())
		err = client.Get(context.Background(), testPodAKey, testPodA)
		Expect(err).ToNot(HaveOccurred())
		Expect(testPodA.GetLabels()["mutated-by"]).To(Equal("a"))

		By("Creating assignMetadata mutation-c")
		amC := &gkv1alpha.AssignMetadata{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mutation-c",
			},
			Spec: gkv1alpha.AssignMetadataSpec{
				Match: gkv1alpha.Match{
					Scope:      apiextensionsv1beta1.NamespaceScoped,
					Namespaces: []string{TestingNamespace},
					Kinds: []gkv1alpha.Kinds{
						{
							APIGroups: []string{""},
							Kinds:     []string{"Pod"},
						},
					},
				},
				Location: "metadata.labels.mutated-by",
				Parameters: gkv1alpha.MetadataParameters{
					Assign: runtime.RawExtension{
						Raw: []byte(fmt.Sprintf("{\"value\":\"c\"}")),
					},
				},
			},
		}
		err = client.Create(context.Background(), amC)
		Expect(err).NotTo(HaveOccurred())

		By("Creating test-pod-c")
		testPodC := pods.DefinePod(TestingNamespace)
		testPodC.SetName("test-pod-c")
		err = client.Create(context.Background(), testPodC)
		Expect(err).NotTo(HaveOccurred())

		By("Asserting that test-pod-c was labeled")
		testPodCKey, err := k8sClient.ObjectKeyFromObject(testPodC)
		Expect(err).ToNot(HaveOccurred())
		err = client.Get(context.Background(), testPodCKey, testPodC)
		Expect(err).ToNot(HaveOccurred())
		Expect(testPodA.GetLabels()["mutated-by"]).To(Equal("a"))
	})

	It("should be able to update mutation policy", func() {
		By("Creating assignMetadata mutation-version with mutation-version: 0")
		am := &gkv1alpha.AssignMetadata{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mutation-version",
			},
			Spec: gkv1alpha.AssignMetadataSpec{
				Match: gkv1alpha.Match{
					Scope:      apiextensionsv1beta1.NamespaceScoped,
					Namespaces: []string{TestingNamespace},
					Kinds: []gkv1alpha.Kinds{
						{
							APIGroups: []string{""},
							Kinds:     []string{"Pod"},
						},
					},
				},
				Location: "metadata.labels.mutation-version",
				Parameters: gkv1alpha.MetadataParameters{
					Assign: runtime.RawExtension{
						Raw: []byte(fmt.Sprintf("{\"value\":\"0\"}")),
					},
				},
			},
		}
		err := client.Create(context.Background(), am)
		Expect(err).NotTo(HaveOccurred())

		By("Creating test-pod-version-0")
		testPodA := pods.DefinePod(TestingNamespace)
		testPodA.SetName("test-pod-version-0")
		err = client.Create(context.Background(), testPodA)
		Expect(err).NotTo(HaveOccurred())

		By("Asserting that test-pod-version-0 was labeled with mutation-version: 0")
		testPodAKey, err := k8sClient.ObjectKeyFromObject(testPodA)
		Expect(err).NotTo(HaveOccurred())
		err = client.Get(context.Background(), testPodAKey, testPodA)
		Expect(err).NotTo(HaveOccurred())
		Expect(testPodA.GetLabels()["mutation-version"]).To(Equal("0"))

		By("Updating assignMetadata to mutation-version: 1")
		newValue := runtime.RawExtension{
			Raw: []byte(fmt.Sprintf("{\"value\":\"1\"}")),
		}
		am.Spec.Parameters.Assign = newValue
		err = client.Update(context.Background(), am)
		Expect(err).ToNot(HaveOccurred())

		By("Asserting that mutation-vesrion was updated to: 1")
		amKey, err := k8sClient.ObjectKeyFromObject(am)
		Expect(err).ToNot(HaveOccurred())
		err = client.Get(context.Background(), amKey, am)
		Expect(err).NotTo(HaveOccurred())
		Expect(am.Spec.Parameters.Assign).To(Equal(newValue))

		By("Creating test-pod-version-1")
		testPodB := pods.DefinePod(TestingNamespace)
		testPodB.SetName("test-pod-version-1")
		err = client.Create(context.Background(), testPodB)
		Expect(err).NotTo(HaveOccurred())

		By("Asserting that test-pod-version-1 was labeled with mutation-version: 1")
		testPodBKey, err := k8sClient.ObjectKeyFromObject(testPodB)
		Expect(err).NotTo(HaveOccurred())
		err = client.Get(context.Background(), testPodBKey, testPodB)
		Expect(err).NotTo(HaveOccurred())
		Expect(testPodB.GetLabels()["mutation-version"]).To(Equal("1"))
	})

	It("should not apply mutations policies after deletion", func() {
		By("Creating the assignMetadata")
		am := &gkv1alpha.AssignMetadata{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mutation-deleted",
			},
			Spec: gkv1alpha.AssignMetadataSpec{
				Match: gkv1alpha.Match{
					Scope:      apiextensionsv1beta1.NamespaceScoped,
					Namespaces: []string{TestingNamespace},
					Kinds: []gkv1alpha.Kinds{
						{
							APIGroups: []string{""},
							Kinds:     []string{"Pod"},
						},
					},
				},
				Location: "metadata.labels.mutated",
				Parameters: gkv1alpha.MetadataParameters{
					Assign: runtime.RawExtension{
						Raw: []byte(fmt.Sprintf("{\"value\":\"true\"}")),
					},
				},
			},
		}
		err := client.Create(context.Background(), am)
		Expect(err).NotTo(HaveOccurred())

		By("Creating pod before-delete")
		podBeforeDelete := pods.DefinePod(TestingNamespace)
		podBeforeDelete.SetName("before-delete")
		err = client.Create(context.Background(), podBeforeDelete)
		Expect(err).NotTo(HaveOccurred())

		By("Asserting that pod before-delete was labeled")
		podBeforeDeleteKey, err := k8sClient.ObjectKeyFromObject(podBeforeDelete)
		Expect(err).ToNot(HaveOccurred())
		err = client.Get(context.Background(), podBeforeDeleteKey, podBeforeDelete)
		Expect(err).ToNot(HaveOccurred())
		Expect(podBeforeDelete.GetLabels()["mutated"]).To(Equal("true"))

		By("Deleting the assignMetadata")
		err = client.Delete(context.Background(), am)
		Expect(err).ToNot(HaveOccurred())

		By("Creating pod after-delete")
		podAfterDelete := pods.DefinePod(TestingNamespace)
		podAfterDelete.SetName("after-delete")
		err = client.Create(context.Background(), podAfterDelete)
		Expect(err).NotTo(HaveOccurred())

		By("Asserting that pod after-delete was not labeld by the deleted policy")
		podAfterDeleteKey, err := k8sClient.ObjectKeyFromObject(podAfterDelete)
		Expect(err).ToNot(HaveOccurred())
		err = client.Get(context.Background(), podAfterDeleteKey, podAfterDelete)
		Expect(err).ToNot(HaveOccurred())
		_, found := podAfterDelete.GetLabels()["mutated"]
		Expect(found).To(BeFalse())
	})

	It("should be able to match by any match category", func() {
		By("Creating the test namespaces")
		includedNamepsace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mutation-included",
				Labels: map[string]string{
					"ns-selected": "true",
				},
			},
		}
		err := client.Create(context.Background(), includedNamepsace)
		Expect(err).NotTo(HaveOccurred())

		excludedNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mutation-excluded",
				Labels: map[string]string{
					"ns-selected": "true",
				},
			},
		}
		err = client.Create(context.Background(), excludedNamespace)
		Expect(err).NotTo(HaveOccurred())

		By("Creating the mutation policies")
		allSelected := &gkv1alpha.AssignMetadata{
			ObjectMeta: metav1.ObjectMeta{
				Name: "all-selected",
			},
			Spec: gkv1alpha.AssignMetadataSpec{
				Match: gkv1alpha.Match{
					Scope: apiextensionsv1beta1.ResourceScope("*"),
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"selected": "true",
						},
					},
				},
				Location: "metadata.labels.all-selected",
				Parameters: gkv1alpha.MetadataParameters{
					Assign: runtime.RawExtension{
						Raw: []byte(fmt.Sprintf("{\"value\":\"true\"}")),
					},
				},
			},
		}
		err = client.Create(context.Background(), allSelected)
		Expect(err).NotTo(HaveOccurred())

		namespaceIncluded := &gkv1alpha.AssignMetadata{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace-included",
			},
			Spec: gkv1alpha.AssignMetadataSpec{
				Match: gkv1alpha.Match{
					Scope:      apiextensionsv1beta1.NamespaceScoped,
					Namespaces: []string{includedNamepsace.GetName()},
				},
				Location: "metadata.labels.namespace-included",
				Parameters: gkv1alpha.MetadataParameters{
					Assign: runtime.RawExtension{
						Raw: []byte(fmt.Sprintf("{\"value\":\"true\"}")),
					},
				},
			},
		}
		err = client.Create(context.Background(), namespaceIncluded)
		Expect(err).NotTo(HaveOccurred())

		clusterSelected := &gkv1alpha.AssignMetadata{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster-selected",
			},
			Spec: gkv1alpha.AssignMetadataSpec{
				Match: gkv1alpha.Match{
					Scope: apiextensionsv1beta1.ClusterScoped,
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"selected": "true",
						},
					},
				},
				Location: "metadata.labels.cluster-selected",
				Parameters: gkv1alpha.MetadataParameters{
					Assign: runtime.RawExtension{
						Raw: []byte(fmt.Sprintf("{\"value\":\"true\"}")),
					},
				},
			},
		}
		err = client.Create(context.Background(), clusterSelected)
		Expect(err).NotTo(HaveOccurred())

		namespaceSelected := &gkv1alpha.AssignMetadata{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace-selected",
			},
			Spec: gkv1alpha.AssignMetadataSpec{
				Match: gkv1alpha.Match{
					Scope: apiextensionsv1beta1.NamespaceScoped,
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"ns-selected": "true",
						},
					},
					ExcludedNamespaces: []string{excludedNamespace.Name},
				},
				Location: "metadata.labels.namespace-selected",
				Parameters: gkv1alpha.MetadataParameters{
					Assign: runtime.RawExtension{
						Raw: []byte(fmt.Sprintf("{\"value\":\"true\"}")),
					},
				},
			},
		}
		err = client.Create(context.Background(), namespaceSelected)
		Expect(err).NotTo(HaveOccurred())

		By("Creating all test objects")
		clusterObject := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "gk-test-object",
				Labels: map[string]string{},
			},
		}
		excludedNamespacedObject := pods.DefinePod(excludedNamespace.GetName())
		includedNamespacedObject := pods.DefinePod(includedNamepsace.GetName())

		By("Initializing cluster object")
		labels := clusterObject.GetLabels()
		labels["selected"] = "true"
		clusterObject.SetLabels(labels)
		err = client.Create(context.Background(), clusterObject)
		Expect(err).ToNot(HaveOccurred())

		By("Initializing excludedNamespacedObject")
		labels = map[string]string{}
		labels["selected"] = "true"
		excludedNamespacedObject.SetLabels(labels)
		err = client.Create(context.Background(), excludedNamespacedObject)
		Expect(err).ToNot(HaveOccurred())

		By("Initializing includedNamespacedObject")
		err = client.Create(context.Background(), includedNamespacedObject)
		Expect(err).ToNot(HaveOccurred())

		By("Asserting that cluster object mutations were applied")
		clusterObjectKey, err := k8sClient.ObjectKeyFromObject(clusterObject)
		Expect(err).ToNot(HaveOccurred())
		err = client.Get(context.Background(), clusterObjectKey, clusterObject)
		Expect(err).ToNot(HaveOccurred())

		labels = clusterObject.GetLabels()

		value, ok := labels["all-selected"]
		Expect(ok).To(Equal(true))
		Expect(value).To(Equal("true"))

		_, ok = labels["namespace-included"]
		Expect(ok).To(Equal(false))

		value, ok = labels["cluster-selected"]
		Expect(ok).To(Equal(true))
		Expect(value).To(Equal("true"))

		_, ok = labels["namespace-selected"]
		Expect(ok).To(Equal(false))

		By("Asserting that excludedNamespaced object mutations were applied")
		excludedNamespacedObjectKey, err := k8sClient.ObjectKeyFromObject(excludedNamespacedObject)
		Expect(err).NotTo(HaveOccurred())
		err = client.Get(context.Background(), excludedNamespacedObjectKey, excludedNamespacedObject)
		Expect(err).NotTo(HaveOccurred())

		labels = excludedNamespacedObject.GetLabels()

		value, ok = labels["all-selected"]
		Expect(ok).To(Equal(true))
		Expect(value).To(Equal("true"))

		_, ok = labels["namespace-included"]
		Expect(ok).To(Equal(false))

		_, ok = labels["cluster-selected"]
		Expect(ok).To(Equal(false))

		_, ok = labels["namespace-selected"]
		Expect(ok).To(Equal(false))

		By("Asserting that includedNamespaced object mutations were applied")
		includedNamespacedObjectKey, err := k8sClient.ObjectKeyFromObject(includedNamespacedObject)
		Expect(err).NotTo(HaveOccurred())
		err = client.Get(context.Background(), includedNamespacedObjectKey, includedNamespacedObject)
		Expect(err).NotTo(HaveOccurred())

		labels = includedNamespacedObject.GetLabels()

		_, ok = labels["all-selected"]
		Expect(ok).To(Equal(false))

		value, ok = labels["namespace-included"]
		Expect(ok).To(Equal(true))
		Expect(value).To(Equal("true"))

		_, ok = labels["cluster-selected"]
		Expect(ok).To(Equal(false))

		value, ok = labels["namespace-selected"]
		Expect(ok).To(Equal(true))
		Expect(value).To(Equal("true"))
	})
})

func deletePods(namespace string, client *testClient.ClientSet) error {
	list := &corev1.PodList{}

	err := client.List(context.Background(), list, &k8sClient.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		return err
	}

	for _, item := range list.Items {
		err = client.Delete(context.Background(), &item)
		if err != nil {
			return err
		}
	}

	return nil
}
