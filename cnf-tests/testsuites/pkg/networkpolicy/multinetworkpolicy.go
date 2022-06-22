package networkpolicy

import (
	"context"

	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	multinetpolicyv1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	client "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
)

type MultiNetworkPolicyOpt func(*multinetpolicyv1.MultiNetworkPolicy)

func MakeMultiNetworkPolicy(targetNetwork string, opts ...MultiNetworkPolicyOpt) *multinetpolicyv1.MultiNetworkPolicy {
	ret := multinetpolicyv1.MultiNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-multinetwork-policy-",
			Annotations: map[string]string{
				"k8s.v1.cni.cncf.io/policy-for": targetNetwork,
			},
		},
	}

	for _, opt := range opts {
		opt(&ret)
	}

	return &ret
}

func WithPodSelector(podSelector metav1.LabelSelector) MultiNetworkPolicyOpt {
	return func(pol *multinetpolicyv1.MultiNetworkPolicy) {
		pol.Spec.PodSelector = podSelector
	}
}

func WithEmptyIngressRules() MultiNetworkPolicyOpt {
	return func(pol *multinetpolicyv1.MultiNetworkPolicy) {
		pol.Spec.PolicyTypes = appendIfNotPresent(pol.Spec.PolicyTypes, multinetpolicyv1.PolicyTypeIngress)
		pol.Spec.Ingress = []multinetpolicyv1.MultiNetworkPolicyIngressRule{}
	}
}

func WithIngressRule(rule multinetpolicyv1.MultiNetworkPolicyIngressRule) MultiNetworkPolicyOpt {
	return func(pol *multinetpolicyv1.MultiNetworkPolicy) {
		pol.Spec.PolicyTypes = appendIfNotPresent(pol.Spec.PolicyTypes, multinetpolicyv1.PolicyTypeIngress)
		pol.Spec.Ingress = append(pol.Spec.Ingress, rule)
	}
}

func WithEmptyEgressRules() MultiNetworkPolicyOpt {
	return func(pol *multinetpolicyv1.MultiNetworkPolicy) {
		pol.Spec.PolicyTypes = appendIfNotPresent(pol.Spec.PolicyTypes, multinetpolicyv1.PolicyTypeEgress)
		pol.Spec.Egress = []multinetpolicyv1.MultiNetworkPolicyEgressRule{}
	}
}

func WithEgressRule(rule multinetpolicyv1.MultiNetworkPolicyEgressRule) MultiNetworkPolicyOpt {
	return func(pol *multinetpolicyv1.MultiNetworkPolicy) {
		pol.Spec.PolicyTypes = appendIfNotPresent(pol.Spec.PolicyTypes, multinetpolicyv1.PolicyTypeEgress)
		pol.Spec.Egress = append(pol.Spec.Egress, rule)
	}
}

func CreateInNamespace(ns string) MultiNetworkPolicyOpt {
	return func(pol *multinetpolicyv1.MultiNetworkPolicy) {
		ret, err := client.Client.MultiNetworkPolicies(ns).
			Create(context.Background(), pol, metav1.CreateOptions{})

		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		ret.DeepCopyInto(pol)
	}
}

func appendIfNotPresent(input []multinetpolicyv1.MultiPolicyType, newElement multinetpolicyv1.MultiPolicyType) []multinetpolicyv1.MultiPolicyType {
	for _, e := range input {
		if e == newElement {
			return input
		}
	}

	return append(input, newElement)
}
