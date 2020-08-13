package discovery

import (
	"context"
	"fmt"
	"os"
	"strconv"

	testclient "github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/nodes"
	perfv1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1"
	sriovv1 "github.com/openshift/sriov-network-operator/pkg/apis/sriovnetwork/v1"
	sriovtestclient "github.com/openshift/sriov-network-operator/test/util/client"
	sriovcluster "github.com/openshift/sriov-network-operator/test/util/cluster"
	corev1 "k8s.io/api/core/v1"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// DpdkResources contains discovered dpdk resources
type DpdkResources struct {
	Profile  *perfv1.PerformanceProfile
	Resource string
	Device   *sriovv1.InterfaceExt
}

// Enabled indicates whether test discovery mode is enabled.
func Enabled() bool {
	discoveryMode, _ := strconv.ParseBool(os.Getenv("DISCOVERY_MODE"))
	return discoveryMode
}

// DiscoverPerformanceProfileAndPolicyWithAvailableNodes finds a profile/sriovPolicy match for which a node with
// allocatable resources is available. It will return a profile/sriovPolicy for a policy with resource name
// "dpdknic", or a pair with the most available resource on node
func DiscoverPerformanceProfileAndPolicyWithAvailableNodes(client *testclient.ClientSet, sriovclient *sriovtestclient.ClientSet, operatorNamespace string, resourceName string, performanceProfiles []*perfv1.PerformanceProfile, nodeSelector map[string]string,
) (discoveredDpdkResources DpdkResources, err error) {
	currentResourceCount := 0
	var sriovInfos *sriovcluster.EnabledNodes
	sriovInfos, err = sriovcluster.DiscoverSriov(sriovclient, operatorNamespace)
	if err != nil {
		return
	}

	sriovPolicies := &sriovv1.SriovNetworkNodePolicyList{}
	err = client.List(context.TODO(), sriovPolicies, &goclient.ListOptions{Namespace: operatorNamespace})
	if err != nil {
		return
	}
	for _, profile := range performanceProfiles {
		profileNodeSelector := nodes.SelectorUnion(nodeSelector, profile.Spec.NodeSelector)
		var nodesAvailable []corev1.Node
		nodesAvailable, err = nodes.AvailableForSelector(profileNodeSelector)
		if err != nil {
			return
		}

		for _, sriovPolicy := range sriovPolicies.Items {
			if sriovPolicy.Spec.DeviceType != "netdevice" {
				continue
			}
			if &sriovPolicy.Spec.NicSelector == nil || &sriovPolicy.Spec.NicSelector.PfNames == nil || len(sriovPolicy.Spec.NicSelector.PfNames) != 1 {
				continue
			}
			for _, node := range nodesAvailable {
				quantity := node.Status.Allocatable[corev1.ResourceName("openshift.io/"+sriovPolicy.Spec.ResourceName)]
				resourceCount64, _ := (&quantity).AsInt64()
				resourceCount := int(resourceCount64)
				var sriovDevice *sriovv1.InterfaceExt
				sriovDevice, err = sriovInfos.FindOneSriovDevice(node.ObjectMeta.Name)
				if err != nil {
					continue
				}
				// Return profile and policy with the prefered resource name if available
				if sriovPolicy.Spec.ResourceName == resourceName {
					discoveredDpdkResources = DpdkResources{profile, sriovPolicy.Spec.ResourceName, sriovDevice}
					return
				}
				if resourceCount > currentResourceCount {
					discoveredDpdkResources = DpdkResources{profile, sriovPolicy.Spec.ResourceName, sriovDevice}
				}
			}
		}
	}
	if currentResourceCount == 0 {
		err = fmt.Errorf("Unable to find a node with available resources")
		return
	}
	return
}
