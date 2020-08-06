package __performance

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
	performancev1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1"

	corev1 "k8s.io/api/core/v1"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
)

var _ = Describe("[rfe_id:27350][performance]Topology Manager", func() {
	var workerRTNodes []corev1.Node
	var profile *performancev1.PerformanceProfile

	BeforeEach(func() {
		var err error
		workerRTNodes, err = nodes.GetByLabels(testutils.NodeSelectorLabels)
		Expect(err).ToNot(HaveOccurred())
		workerRTNodes, err = nodes.MatchingOptionalSelector(workerRTNodes)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error looking for the optional selector: %v", err))
		Expect(workerRTNodes).ToNot(BeEmpty())
		profile, err = profiles.GetByNodeLabels(testutils.NodeSelectorLabels)
		Expect(err).ToNot(HaveOccurred())
	})

	It("[test_id:26932][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] should be enabled with the policy specified in profile", func() {
		kubeletConfig, err := nodes.GetKubeletConfig(&workerRTNodes[0])
		Expect(err).ToNot(HaveOccurred())

		// verify topology manager policy
		if profile.Spec.NUMA != nil && profile.Spec.NUMA.TopologyPolicy != nil {
			Expect(kubeletConfig.TopologyManagerPolicy).To(Equal(*profile.Spec.NUMA.TopologyPolicy))
		} else {
			Expect(kubeletConfig.TopologyManagerPolicy).To(Equal(kubeletconfigv1beta1.BestEffortTopologyManagerPolicy))
		}
	})
})
