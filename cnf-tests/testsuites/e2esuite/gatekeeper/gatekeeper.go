package gatekeeper

import (
	"context"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testClient "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/pods"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"

	gkopv1alpha "github.com/gatekeeper/gatekeeper-operator/api/v1alpha1"
	gkv1alpha "github.com/open-policy-agent/gatekeeper/apis/mutations/v1alpha1"
	gkmatch "github.com/open-policy-agent/gatekeeper/pkg/mutation/match"
	gktypes "github.com/open-policy-agent/gatekeeper/pkg/mutation/types"
	gkutil "github.com/open-policy-agent/gatekeeper/pkg/util"

	admission "k8s.io/api/admissionregistration/v1"
)

const (
	testingNamespace          = utils.GatekeeperTestingNamespace
	mutationIncludedNamespace = utils.GatekeeperMutationIncludedNamespace
	mutationExcludedNamespace = utils.GatekeeperMutationExcludedNamespace
	mutationEnabledNamespace  = utils.GatekeeperMutationEnabledNamespace
	mutationDisabledNamespace = utils.GatekeeperMutationDisabledNamespace
	testObjectNamespace       = utils.GatekeeperTestObjectNamespace
)

var _ = Describe("gatekeeper", func() {
	client := testClient.Client

	AfterEach(func() {
		err := deletePods(testingNamespace, client)
		Expect(err).NotTo(HaveOccurred())

		namespacesUsed := []string{mutationIncludedNamespace, mutationExcludedNamespace, mutationEnabledNamespace, mutationDisabledNamespace, testObjectNamespace}

		for _, namespace := range namespacesUsed {
			if namespaces.Exists(namespace, client) {
				err := client.Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
		}

		amList := &gkv1alpha.AssignMetadataList{}

		err = client.List(context.Background(), amList)
		Expect(err).NotTo(HaveOccurred())

		for _, am := range amList.Items {
			err := client.Delete(context.Background(), &am)
			Expect(err).NotTo(HaveOccurred())
		}

		for _, namespace := range namespacesUsed {
			err := namespaces.WaitForDeletion(client, namespace, time.Minute)
			Expect(err).ToNot(HaveOccurred())
		}

		gkConfig := &gkopv1alpha.Gatekeeper{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gatekeeper",
			},
		}

		gkConfigKey := k8sClient.ObjectKeyFromObject(gkConfig)
		Expect(err).ToNot(HaveOccurred())

		err = client.Get(context.Background(), gkConfigKey, gkConfig)
		Expect(err).ToNot(HaveOccurred())

		if gkConfig.Spec.Webhook != nil && gkConfig.Spec.Webhook.NamespaceSelector != nil {
			gkConfig.Spec.Webhook.NamespaceSelector = nil
			err = client.Update(context.Background(), gkConfig, &k8sClient.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}
	})

	Context("mutation", func() {
		It("should be able to add metadata info(labels/annotations)",
			func() {
				amList := []*gkv1alpha.AssignMetadata{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "add-label",
						},
						Spec: gkv1alpha.AssignMetadataSpec{
							Match: gkmatch.Match{
								Scope:      apiextensionsv1.NamespaceScoped,
								Namespaces: []gkutil.PrefixWildcard{testingNamespace},
								Kinds: []gkmatch.Kinds{
									{
										APIGroups: []string{""},
										Kinds:     []string{"Pod"},
									},
								},
							},
							Location: "metadata.labels.mutated",
							Parameters: gkv1alpha.MetadataParameters{
								Assign: gkv1alpha.AssignField{
									Value: &gktypes.Anything{
										Value: "true",
									},
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "add-annotation",
						},
						Spec: gkv1alpha.AssignMetadataSpec{
							Match: gkmatch.Match{
								Scope:      apiextensionsv1.NamespaceScoped,
								Namespaces: []gkutil.PrefixWildcard{testingNamespace},
								Kinds: []gkmatch.Kinds{
									{
										APIGroups: []string{""},
										Kinds:     []string{"Pod"},
									},
								},
							},
							Location: "metadata.annotations.mutated",
							Parameters: gkv1alpha.MetadataParameters{
								Assign: gkv1alpha.AssignField{
									Value: &gktypes.Anything{
										Value: "true",
									},
								},
							},
						},
					},
				}

				for _, am := range amList {
					err := client.Create(context.Background(), am)
					Expect(err).NotTo(HaveOccurred())
				}

				pod := pods.DefinePod(testingNamespace)
				err := client.Create(context.Background(), pod)
				Expect(err).NotTo(HaveOccurred())

				podKey := k8sClient.ObjectKeyFromObject(pod)
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
							Match: gkmatch.Match{
								Scope:      apiextensionsv1.NamespaceScoped,
								Namespaces: []gkutil.PrefixWildcard{testingNamespace},
								Kinds: []gkmatch.Kinds{
									{
										APIGroups: []string{""},
										Kinds:     []string{"Pod"},
									},
								},
							},
							Location: "metadata.labels.mutated",
							Parameters: gkv1alpha.MetadataParameters{
								Assign: gkv1alpha.AssignField{
									Value: &gktypes.Anything{
										Value: "true",
									},
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "mutate-annotation",
						},
						Spec: gkv1alpha.AssignMetadataSpec{
							Match: gkmatch.Match{
								Scope:      apiextensionsv1.NamespaceScoped,
								Namespaces: []gkutil.PrefixWildcard{testingNamespace},
								Kinds: []gkmatch.Kinds{
									{
										APIGroups: []string{""},
										Kinds:     []string{"Pod"},
									},
								},
							},
							Location: "metadata.annotations.mutated",
							Parameters: gkv1alpha.MetadataParameters{
								Assign: gkv1alpha.AssignField{
									Value: &gktypes.Anything{
										Value: "true",
									},
								},
							},
						},
					},
				}

				for _, am := range amList {
					err := client.Create(context.Background(), am)
					Expect(err).NotTo(HaveOccurred())
				}

				pod := pods.DefinePod(testingNamespace)
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

				podKey := k8sClient.ObjectKeyFromObject(pod)
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
					Match: gkmatch.Match{
						Scope:      apiextensionsv1.NamespaceScoped,
						Namespaces: []gkutil.PrefixWildcard{testingNamespace},
						Kinds: []gkmatch.Kinds{
							{
								APIGroups: []string{""},
								Kinds:     []string{"Pod"},
							},
						},
					},
					Location: "metadata.labels.mutated-by",
					Parameters: gkv1alpha.MetadataParameters{
						Assign: gkv1alpha.AssignField{
							Value: &gktypes.Anything{
								Value: "b",
							},
						},
					},
				},
			}
			err := client.Create(context.Background(), amB)
			Expect(err).NotTo(HaveOccurred())

			By("Creating test-pod-b")
			testPodB := pods.DefinePod(testingNamespace)
			testPodB.SetName("test-pod-b")
			err = client.Create(context.Background(), testPodB)
			Expect(err).NotTo(HaveOccurred())

			By("Asserting that test-pod-b was labeled")
			testPodBKey := k8sClient.ObjectKeyFromObject(testPodB)
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
					Match: gkmatch.Match{
						Scope:      apiextensionsv1.NamespaceScoped,
						Namespaces: []gkutil.PrefixWildcard{testingNamespace},
						Kinds: []gkmatch.Kinds{
							{
								APIGroups: []string{""},
								Kinds:     []string{"Pod"},
							},
						},
					},
					Location: "metadata.labels.mutated-by",
					Parameters: gkv1alpha.MetadataParameters{
						Assign: gkv1alpha.AssignField{
							Value: &gktypes.Anything{
								Value: "a",
							},
						},
					},
				},
			}
			err = client.Create(context.Background(), amA)
			Expect(err).NotTo(HaveOccurred())

			By("Creating test-pod-a")
			testPodA := pods.DefinePod(testingNamespace)
			testPodA.SetName("test-pod-a")
			err = client.Create(context.Background(), testPodA)
			Expect(err).NotTo(HaveOccurred())

			By("Asserting that test-pod-a was labeled")
			testPodAKey := k8sClient.ObjectKeyFromObject(testPodA)
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
					Match: gkmatch.Match{
						Scope:      apiextensionsv1.NamespaceScoped,
						Namespaces: []gkutil.PrefixWildcard{testingNamespace},
						Kinds: []gkmatch.Kinds{
							{
								APIGroups: []string{""},
								Kinds:     []string{"Pod"},
							},
						},
					},
					Location: "metadata.labels.mutated-by",
					Parameters: gkv1alpha.MetadataParameters{
						Assign: gkv1alpha.AssignField{
							Value: &gktypes.Anything{
								Value: "c",
							},
						},
					},
				},
			}
			err = client.Create(context.Background(), amC)
			Expect(err).NotTo(HaveOccurred())

			By("Creating test-pod-c")
			testPodC := pods.DefinePod(testingNamespace)
			testPodC.SetName("test-pod-c")
			err = client.Create(context.Background(), testPodC)
			Expect(err).NotTo(HaveOccurred())

			By("Asserting that test-pod-c was labeled")
			testPodCKey := k8sClient.ObjectKeyFromObject(testPodC)
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
					Match: gkmatch.Match{
						Scope:      apiextensionsv1.NamespaceScoped,
						Namespaces: []gkutil.PrefixWildcard{testingNamespace},
						Kinds: []gkmatch.Kinds{
							{
								APIGroups: []string{""},
								Kinds:     []string{"Pod"},
							},
						},
					},
					Location: "metadata.labels.mutation-version",
					Parameters: gkv1alpha.MetadataParameters{
						Assign: gkv1alpha.AssignField{
							Value: &gktypes.Anything{
								Value: "0",
							},
						},
					},
				},
			}
			err := client.Create(context.Background(), am)
			Expect(err).NotTo(HaveOccurred())

			By("Creating test-pod-version-0")
			testPodA := pods.DefinePod(testingNamespace)
			testPodA.SetName("test-pod-version-0")
			err = client.Create(context.Background(), testPodA)
			Expect(err).NotTo(HaveOccurred())

			By("Asserting that test-pod-version-0 was labeled with mutation-version: 0")
			testPodAKey := k8sClient.ObjectKeyFromObject(testPodA)
			Expect(err).NotTo(HaveOccurred())
			err = client.Get(context.Background(), testPodAKey, testPodA)
			Expect(err).NotTo(HaveOccurred())
			Expect(testPodA.GetLabels()["mutation-version"]).To(Equal("0"))

			By("Updating assignMetadata to mutation-version: 1")
			newValue := gkv1alpha.AssignField{
				Value: &gktypes.Anything{
					Value: "1",
				},
			}

			client.Get(context.Background(), k8sClient.ObjectKeyFromObject(am), am)
			am.Spec.Parameters.Assign = newValue
			err = client.Update(context.Background(), am)
			Expect(err).ToNot(HaveOccurred())

			By("Asserting that mutation-vesrion was updated to: 1")
			amKey := k8sClient.ObjectKeyFromObject(am)
			Expect(err).ToNot(HaveOccurred())
			err = client.Get(context.Background(), amKey, am)
			Expect(err).NotTo(HaveOccurred())
			Expect(am.Spec.Parameters.Assign).To(Equal(newValue))

			By("Creating test-pod-version-1")
			testPodB := pods.DefinePod(testingNamespace)
			testPodB.SetName("test-pod-version-1")
			err = client.Create(context.Background(), testPodB)
			Expect(err).NotTo(HaveOccurred())

			By("Asserting that test-pod-version-1 was labeled with mutation-version: 1")
			testPodBKey := k8sClient.ObjectKeyFromObject(testPodB)
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
					Match: gkmatch.Match{
						Scope:      apiextensionsv1.NamespaceScoped,
						Namespaces: []gkutil.PrefixWildcard{testingNamespace},
						Kinds: []gkmatch.Kinds{
							{
								APIGroups: []string{""},
								Kinds:     []string{"Pod"},
							},
						},
					},
					Location: "metadata.labels.mutated",
					Parameters: gkv1alpha.MetadataParameters{
						Assign: gkv1alpha.AssignField{
							Value: &gktypes.Anything{
								Value: "true",
							},
						},
					},
				},
			}
			err := client.Create(context.Background(), am)
			Expect(err).NotTo(HaveOccurred())

			By("Creating pod before-delete")
			podBeforeDelete := pods.DefinePod(testingNamespace)
			podBeforeDelete.SetName("before-delete")
			err = client.Create(context.Background(), podBeforeDelete)
			Expect(err).NotTo(HaveOccurred())

			By("Asserting that pod before-delete was labeled")
			podBeforeDeleteKey := k8sClient.ObjectKeyFromObject(podBeforeDelete)
			Expect(err).ToNot(HaveOccurred())
			err = client.Get(context.Background(), podBeforeDeleteKey, podBeforeDelete)
			Expect(err).ToNot(HaveOccurred())
			Expect(podBeforeDelete.GetLabels()["mutated"]).To(Equal("true"))

			By("Deleting the assignMetadata")
			err = client.Delete(context.Background(), am)
			Expect(err).ToNot(HaveOccurred())

			By("Creating pod after-delete")
			podAfterDelete := pods.DefinePod(testingNamespace)
			podAfterDelete.SetName("after-delete")
			err = client.Create(context.Background(), podAfterDelete)
			Expect(err).NotTo(HaveOccurred())

			By("Asserting that pod after-delete was not labeld by the deleted policy")
			podAfterDeleteKey := k8sClient.ObjectKeyFromObject(podAfterDelete)
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
					Match: gkmatch.Match{
						Scope: apiextensionsv1.ResourceScope("*"),
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"selected": "true",
							},
						},
					},
					Location: "metadata.labels.all-selected",
					Parameters: gkv1alpha.MetadataParameters{
						Assign: gkv1alpha.AssignField{
							Value: &gktypes.Anything{
								Value: "true",
							},
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
					Match: gkmatch.Match{
						Scope:      apiextensionsv1.NamespaceScoped,
						Namespaces: []gkutil.PrefixWildcard{gkutil.PrefixWildcard(includedNamepsace.GetName())},
					},
					Location: "metadata.labels.namespace-included",
					Parameters: gkv1alpha.MetadataParameters{
						Assign: gkv1alpha.AssignField{
							Value: &gktypes.Anything{
								Value: "true",
							},
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
					Match: gkmatch.Match{
						Scope: apiextensionsv1.ClusterScoped,
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"selected": "true",
							},
						},
					},
					Location: "metadata.labels.cluster-selected",
					Parameters: gkv1alpha.MetadataParameters{
						Assign: gkv1alpha.AssignField{
							Value: &gktypes.Anything{
								Value: "true",
							},
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
					Match: gkmatch.Match{
						Scope: apiextensionsv1.NamespaceScoped,
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"ns-selected": "true",
							},
						},

						ExcludedNamespaces: []gkutil.PrefixWildcard{gkutil.PrefixWildcard(excludedNamespace.Name)},
					},
					Location: "metadata.labels.namespace-selected",
					Parameters: gkv1alpha.MetadataParameters{
						Assign: gkv1alpha.AssignField{
							Value: &gktypes.Anything{
								Value: "true",
							},
						},
					},
				},
			}
			err = client.Create(context.Background(), namespaceSelected)
			Expect(err).NotTo(HaveOccurred())

			for _, am := range []*gkv1alpha.AssignMetadata{allSelected, namespaceIncluded, clusterSelected, namespaceSelected} {
				Eventually(func() bool {

					getAm := &gkv1alpha.AssignMetadata{}
					err := client.Get(context.Background(), k8sClient.ObjectKeyFromObject(am), getAm)
					Expect(err).ToNot(HaveOccurred())
					podStatuses := getAm.Status.ByPod
					for _, podStatus := range podStatuses {
						if !podStatus.Enforced {
							return false
						}
					}
					return true
				}, 10*time.Second, 2*time.Second).Should(Equal(true), "Mutations are not applied")
			}

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
			clusterObjectKey := k8sClient.ObjectKeyFromObject(clusterObject)
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
			excludedNamespacedObjectKey := k8sClient.ObjectKeyFromObject(excludedNamespacedObject)
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
			includedNamespacedObjectKey := k8sClient.ObjectKeyFromObject(includedNamespacedObject)
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

	Context("operator", func() {
		It("should be able to select mutation namespaces", func() {
			var err error

			By("Adding namespace selector to gatekeeper operator config")
			gkConfig := &gkopv1alpha.Gatekeeper{
				ObjectMeta: metav1.ObjectMeta{
					Name: "gatekeeper",
				},
				Spec: gkopv1alpha.GatekeeperSpec{
					Webhook: &gkopv1alpha.WebhookConfig{},
				},
			}

			gkConfigKey := k8sClient.ObjectKeyFromObject(gkConfig)
			Expect(err).ToNot(HaveOccurred())

			err = client.Get(context.Background(), gkConfigKey, gkConfig)
			Expect(err).ToNot(HaveOccurred())

			gkConfig.Spec.Webhook.NamespaceSelector = &metav1.LabelSelector{
				MatchLabels: map[string]string{"mutate": "enabled"},
			}

			err = client.Update(context.Background(), gkConfig, &k8sClient.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())

			mutatinWebhookConfiguration := &admission.MutatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "gatekeeper-mutating-webhook-configuration",
				},
			}
			mwConfigKey := k8sClient.ObjectKeyFromObject(mutatinWebhookConfiguration)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				err := client.Get(context.Background(), mwConfigKey, mutatinWebhookConfiguration)
				Expect(err).ToNot(HaveOccurred())
				// Webhook must update its namespaceSelector to match gatekeeper namespaceSelector
				return len(mutatinWebhookConfiguration.Webhooks) == 1 &&
					mutatinWebhookConfiguration.Webhooks[0].NamespaceSelector != nil &&
					reflect.DeepEqual(mutatinWebhookConfiguration.Webhooks[0].NamespaceSelector.MatchLabels, gkConfig.Spec.Webhook.NamespaceSelector.MatchLabels)
			}, 1*time.Minute, 1*time.Second).Should(BeTrue())

			By("Creating an all pod mutation")
			podMutation := &gkv1alpha.AssignMetadata{
				ObjectMeta: metav1.ObjectMeta{
					Name: "all-pod-mutation",
				},
				Spec: gkv1alpha.AssignMetadataSpec{
					Match: gkmatch.Match{
						Scope: apiextensionsv1.NamespaceScoped,
						Kinds: []gkmatch.Kinds{
							{
								APIGroups: []string{""},
								Kinds:     []string{"Pod"},
							},
						},
					},
					Location: "metadata.labels.mutated",
					Parameters: gkv1alpha.MetadataParameters{
						Assign: gkv1alpha.AssignField{
							Value: &gktypes.Anything{
								Value: "true",
							},
						},
					},
				},
			}
			err = client.Create(context.Background(), podMutation)
			Expect(err).NotTo(HaveOccurred())

			By("Creating the test namespaces")
			mutationEnabledNamespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: mutationEnabledNamespace,
					Labels: map[string]string{
						"mutate": "enabled",
					},
				},
			}
			err = client.Create(context.Background(), mutationEnabledNamespace)
			Expect(err).NotTo(HaveOccurred())

			mutationDisabledNamespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: mutationDisabledNamespace,
				},
			}
			err = client.Create(context.Background(), mutationDisabledNamespace)
			Expect(err).NotTo(HaveOccurred())

			By("Creating the test pods")
			mutationEnabledPod := pods.DefinePod(mutationEnabledNamespace.GetName())
			err = client.Create(context.Background(), mutationEnabledPod)
			Expect(err).NotTo(HaveOccurred())

			mutationDisabledPod := pods.DefinePod(mutationDisabledNamespace.GetName())
			err = client.Create(context.Background(), mutationDisabledPod)
			Expect(err).NotTo(HaveOccurred())

			By("Asserting that only the pods in selected namespaces were mutated")
			mutationEnabledPod, err = client.Pods(mutationEnabledNamespace.GetName()).Get(context.Background(), mutationEnabledPod.GetName(), metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			value, ok := mutationEnabledPod.Labels["mutated"]
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal("true"))

			mutationDisabledPod, err = client.Pods(mutationDisabledNamespace.GetName()).Get(context.Background(), mutationDisabledPod.GetName(), metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			_, ok = mutationDisabledPod.Labels["mutated"]
			Expect(ok).To(BeFalse())
		})
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
