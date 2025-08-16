package performanceprofile

import (
	"context"
	"fmt"
	"strings"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/machineconfigpool"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/cpuset"

	"k8s.io/apimachinery/pkg/api/errors"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	performancev2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"
)

const (
	Arm64KPerformanceProfileHugepageSize = "512M"
	Arm64KHugepageSize                   = "524288kB"
	X86PerformanceProfileHugepageSize    = "1G"
	X86HugepageSize                      = "1048576kB"
)

var (
	OriginalPerformanceProfile *performancev2.PerformanceProfile
	HugePageSize               string
)

func FindDefaultPerformanceProfile(performanceProfileName string) (*performancev2.PerformanceProfile, error) {
	performanceProfile := &performancev2.PerformanceProfile{}
	err := client.Client.Get(context.TODO(), goclient.ObjectKey{Name: performanceProfileName}, performanceProfile)
	return performanceProfile, err
}

func FindOrOverridePerformanceProfile(performanceProfileName, machineConfigPoolName string) error {
	var valid = true
	performanceProfile, err := FindDefaultPerformanceProfile(performanceProfileName)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		valid = false
		performanceProfile = nil
	}
	if valid {
		valid, err = ValidatePerformanceProfile(performanceProfile)
		if err != nil {
			return err
		}
	}
	if !valid {
		mcp := &mcv1.MachineConfigPool{}
		err = client.Client.Get(context.TODO(), goclient.ObjectKey{Name: machineConfigPoolName}, mcp)
		if err != nil {
			return err
		}

		if performanceProfile != nil {
			OriginalPerformanceProfile = performanceProfile.DeepCopy()

			// Clean and create a new performance profile for the dpdk application
			err = CleanPerformanceProfiles()
			if err != nil {
				return err
			}

			err = machineconfigpool.WaitForMCPStable(*mcp)
			if err != nil {
				return err
			}
		}

		err = CreatePerformanceProfile(performanceProfileName, mcp)
		if err != nil {
			return err
		}

		err = machineconfigpool.WaitForMCPStable(*mcp)
		if err != nil {
			return err
		}
	}

	return nil
}

func ValidatePerformanceProfile(performanceProfile *performancev2.PerformanceProfile) (bool, error) {
	// Check we have more then two isolated CPU
	cpuSet, err := cpuset.Parse(string(*performanceProfile.Spec.CPU.Isolated))
	if err != nil {
		return false, err
	}

	cpuSetSlice := cpuSet.List()
	if len(cpuSetSlice) < 6 {
		return false, nil
	}

	if performanceProfile.Spec.HugePages == nil {
		return false, nil
	}

	if len(performanceProfile.Spec.HugePages.Pages) == 0 {
		return false, nil
	}

	foundHugePages := false
	for _, page := range performanceProfile.Spec.HugePages.Pages {
		if page.Size == X86PerformanceProfileHugepageSize {
			countVerification := 5
			// we need a minimum of 5 huge pages so if there is no Node in the performance profile we need 10 pages
			// because the kernel will split the number in the performance policy equally to all the numa's
			if page.Node == nil {
				countVerification = countVerification * 2
			}

			if page.Count < int32(countVerification) {
				continue
			}

			foundHugePages = true
			HugePageSize = X86PerformanceProfileHugepageSize
			break
		} else if page.Size == Arm64KPerformanceProfileHugepageSize {
			countVerification := 4

			// TODO: we need to handle this in the future by checking how many numas exist on a node to calculate the number of hugepages needed
			//if page.Node == nil {
			//	countVerification = countVerification * 2
			//}

			if page.Count < int32(countVerification) {
				continue
			}

			foundHugePages = true
			HugePageSize = Arm64KPerformanceProfileHugepageSize
			break
		}
	}

	return foundHugePages, nil
}

func DiscoverPerformanceProfiles(enforcedPerformanceProfileName string) (bool, string, []*performancev2.PerformanceProfile) {
	if enforcedPerformanceProfileName != "" {
		performanceProfile, err := FindDefaultPerformanceProfile(enforcedPerformanceProfileName)
		if err != nil {
			return false, fmt.Sprintf("Can not run tests in discovery mode. Failed to find a valid perfomance profile. %s", err), nil
		}
		valid, err := ValidatePerformanceProfile(performanceProfile)
		if !valid || err != nil {
			return false, fmt.Sprintf("Can not run tests in discovery mode. Failed to find a valid perfomance profile. %s", err), nil
		}
		return true, "", []*performancev2.PerformanceProfile{performanceProfile}
	}

	performanceProfileList := &performancev2.PerformanceProfileList{}
	var profiles []*performancev2.PerformanceProfile
	err := client.Client.List(context.TODO(), performanceProfileList)
	if err != nil {
		return false, fmt.Sprintf("Can not run tests in discovery mode. Failed to find a valid perfomance profile. %s", err), nil
	}
	for _, performanceProfile := range performanceProfileList.Items {
		valid, err := ValidatePerformanceProfile(&performanceProfile)
		if valid && err == nil {
			profiles = append(profiles, &performanceProfile)
		}
	}
	if len(profiles) > 0 {
		return true, "", profiles
	}
	return false, fmt.Sprintf("Can not run tests in discovery mode. Failed to find a valid perfomance profile. %s", err), nil
}

func CreatePerformanceProfile(performanceProfileName string, machineConfigPool *mcv1.MachineConfigPool) error {
	nodes := &corev1.NodeList{}
	err := client.Client.List(context.TODO(), nodes, &goclient.ListOptions{LabelSelector: labels.SelectorFromSet(machineConfigPool.Spec.NodeSelector.MatchLabels)})
	if err != nil {
		return err
	}

	if len(nodes.Items) == 0 {
		return fmt.Errorf("Failed to find nodes for machine config pool: %s", machineConfigPool.Name)
	}

	isolatedCPUSet := performancev2.CPUSet("8-15")
	reservedCPUSet := performancev2.CPUSet("0-7")
	performanceProfile := &performancev2.PerformanceProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name: performanceProfileName,
		},
		Spec: performancev2.PerformanceProfileSpec{
			CPU: &performancev2.CPU{
				Isolated: &isolatedCPUSet,
				Reserved: &reservedCPUSet,
			},
			NodeSelector: map[string]string{
				fmt.Sprintf("node-role.kubernetes.io/%s", machineConfigPool.Name): "",
			},
		},
	}

	// TODO: this will not work for clusters containing both X86 and ARM systems
	// TODO: But we can always use node selector to select only one type on nodes
	if nodes.Items[0].Status.NodeInfo.Architecture == "amd64" {
		hugepageSize := performancev2.HugePageSize(X86PerformanceProfileHugepageSize)
		performanceProfile.Spec.HugePages = &performancev2.HugePages{
			DefaultHugePagesSize: &hugepageSize,
			Pages: []performancev2.HugePage{
				{
					Count: int32(10),
					Size:  hugepageSize,
				},
			},
		}

	} else if nodes.Items[0].Status.NodeInfo.Architecture == "arm64" {
		if !strings.Contains(nodes.Items[0].Status.NodeInfo.KernelVersion, "aarch64+64k") {
			return fmt.Errorf("we only support kernel page size of 64k for ARM systems")
		}
		hugepageSize := performancev2.HugePageSize(Arm64KPerformanceProfileHugepageSize)
		performanceProfile.Spec.HugePages = &performancev2.HugePages{
			DefaultHugePagesSize: &hugepageSize,
			Pages: []performancev2.HugePage{
				{
					Count: int32(32),
					Size:  hugepageSize,
				},
			},
		}

		// we need to also add the annotation to support this system on kubelet
		performanceProfile.Annotations = map[string]string{"kubeletconfig.experimental": `{"topologyManagerPolicyOptions": {"max-allowable-numa-nodes":"16"}}`}

	} else {
		return fmt.Errorf("unsupported system")
	}

	// If the machineConfigPool is master, the automatic selector from PAO won't work
	// since the machineconfiguration.openshift.io/role label is not applied to the
	// master pool, hence we put an explicit selector here.
	if machineConfigPool.Name == "master" {
		performanceProfile.Spec.MachineConfigPoolSelector = map[string]string{
			"pools.operator.machineconfiguration.openshift.io/master": "",
		}
	}

	return client.Client.Create(context.TODO(), performanceProfile)
}

func CleanPerformanceProfiles() error {
	performanceProfileList := &performancev2.PerformanceProfileList{}
	err := client.Client.List(context.TODO(), performanceProfileList, &goclient.ListOptions{})
	if err != nil {
		return err
	}

	for _, policy := range performanceProfileList.Items {
		err := client.Client.Delete(context.TODO(), &policy, &goclient.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func RestorePerformanceProfile(machineConfigPoolName string) error {
	if OriginalPerformanceProfile == nil {
		return nil
	}

	err := CleanPerformanceProfiles()
	if err != nil {
		return err
	}

	mcp := &mcv1.MachineConfigPool{}
	err = client.Client.Get(context.TODO(), goclient.ObjectKey{Name: machineConfigPoolName}, mcp)
	if err != nil {
		return err
	}

	err = machineconfigpool.WaitForMCPStable(*mcp)
	if err != nil {
		return err
	}

	name := OriginalPerformanceProfile.Name
	OriginalPerformanceProfile.ObjectMeta = metav1.ObjectMeta{Name: name}
	err = client.Client.Create(context.TODO(), OriginalPerformanceProfile)
	if err != nil {
		return err
	}

	err = machineconfigpool.WaitForMCPStable(*mcp)
	return err
}

func IsSingleNUMANode(perfProfile *performancev2.PerformanceProfile) bool {
	if perfProfile.Spec.NUMA == nil {
		return false
	}

	if perfProfile.Spec.NUMA.TopologyPolicy == nil {
		return false
	}

	return *perfProfile.Spec.NUMA.TopologyPolicy == "single-numa-node"
}
