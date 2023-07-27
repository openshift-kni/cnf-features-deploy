package performanceprofile

import (
	"context"
	"fmt"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/machineconfigpool"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"

	"k8s.io/apimachinery/pkg/api/errors"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	performancev2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"
)

var (
	OriginalPerformanceProfile *performancev2.PerformanceProfile
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

		err = CreatePerformanceProfile(performanceProfileName, machineConfigPoolName)
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

	cpuSetSlice := cpuSet.ToSlice()
	if len(cpuSetSlice) < 6 {
		return false, nil
	}

	if performanceProfile.Spec.HugePages == nil {
		return false, nil
	}

	if len(performanceProfile.Spec.HugePages.Pages) == 0 {
		return false, nil
	}

	found1GHugePages := false
	for _, page := range performanceProfile.Spec.HugePages.Pages {
		countVerification := 5
		// we need a minimum of 5 huge pages so if there is no Node in the performance profile we need 10 pages
		// because the kernel will split the number in the performance policy equally to all the numa's
		if page.Node == nil {
			countVerification = countVerification * 2
		}

		if page.Size != "1G" {
			continue
		}

		if page.Count < int32(countVerification) {
			continue
		}

		found1GHugePages = true
		break
	}

	return found1GHugePages, nil
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

func CreatePerformanceProfile(performanceProfileName, machineConfigPoolName string) error {
	isolatedCPUSet := performancev2.CPUSet("8-15")
	reservedCPUSet := performancev2.CPUSet("0-7")
	hugepageSize := performancev2.HugePageSize("1G")
	performanceProfile := &performancev2.PerformanceProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name: performanceProfileName,
		},
		Spec: performancev2.PerformanceProfileSpec{
			CPU: &performancev2.CPU{
				Isolated: &isolatedCPUSet,
				Reserved: &reservedCPUSet,
			},
			HugePages: &performancev2.HugePages{
				DefaultHugePagesSize: &hugepageSize,
				Pages: []performancev2.HugePage{
					{
						Count: 10,
						Size:  hugepageSize,
					},
				},
			},
			NodeSelector: map[string]string{
				fmt.Sprintf("node-role.kubernetes.io/%s", machineConfigPoolName): "",
			},
		},
	}

	// If the machineConfigPool is master, the automatic selector from PAO won't work
	// since the machineconfiguration.openshift.io/role label is not applied to the
	// master pool, hence we put an explicit selector here.
	if machineConfigPoolName == "master" {
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
