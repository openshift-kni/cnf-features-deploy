package __performance_config

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	"github.com/openshift-kni/performance-addon-operators/functests/utils"
	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/mcps"
	"github.com/openshift-kni/performance-addon-operators/pkg/apis"
	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"

	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/profile"
)

var _ = Describe("[performance][config] Performance configuration", func() {

	It("Should successfully deploy the performance profile", func() {

		performanceProfile := testProfile()
		performanceManifest, found := os.LookupEnv("PERFORMANCE_PROFILE_MANIFEST_OVERRIDE")
		if found {
			var err error
			performanceProfile, err = externalPerformanceProfile(performanceManifest)
			Expect(err).ToNot(HaveOccurred(), "Failed overriding performance profile", performanceManifest)
			klog.Warning("Consuming performance profile from ", performanceManifest)
		}

		By("Creating the PerformanceProfile")
		// this might fail while the operator is still being deployed and the CRD does not exist yet
		profileAlreadyExists := false
		Eventually(func() error {
			err := testclient.Client.Create(context.TODO(), performanceProfile)
			if errors.IsAlreadyExists(err) {
				klog.Warning(fmt.Sprintf("A PerformanceProfile with name %s already exists! If created externally, tests might have unexpected behaviour", performanceProfile.Name))
				profileAlreadyExists = true
				return nil
			}
			return err
		}, 15*time.Minute, 15*time.Second).ShouldNot(HaveOccurred(), "Failed creating the performance profile")

		By("Getting MCP for profile")
		mcpLabel := profile.GetMachineConfigLabel(performanceProfile)
		key, value := components.GetFirstKeyAndValue(mcpLabel)
		mcpsByLabel, err := mcps.GetByLabel(key, value)
		Expect(err).ToNot(HaveOccurred(), "Failed getting MCP")
		Expect(len(mcpsByLabel)).To(Equal(1), fmt.Sprintf("Unexpected number of MCPs found: %v", len(mcpsByLabel)))
		performanceMCP := &mcpsByLabel[0]

		if !performanceMCP.Spec.Paused {
			By("MCP is already unpaused")
		} else {
			By("Unpausing the MCP")
			Expect(testclient.Client.Patch(context.TODO(), performanceMCP,
				client.ConstantPatch(
					types.JSONPatchType,
					[]byte(fmt.Sprintf(`[{ "op": "replace", "path": "/spec/paused", "value": %v }]`, false)),
				),
			)).ToNot(HaveOccurred(), "Failed unpausing MCP")
		}

		By("Waiting for the MCP to pick the PerformanceProfile's MC")
		mcps.WaitForProfilePickedUp(performanceMCP.Name, performanceProfile.Name)

		// If the profile is already there, it's likely to have been through the updating phase, so we only
		// wait for updated.
		if !profileAlreadyExists {
			By("Waiting for MCP starting to update")
			mcps.WaitForCondition(performanceMCP.Name, mcv1.MachineConfigPoolUpdating, corev1.ConditionTrue)
		}
		By("Waiting for MCP being updated")
		mcps.WaitForCondition(performanceMCP.Name, mcv1.MachineConfigPoolUpdated, corev1.ConditionTrue)

	})

})

func externalPerformanceProfile(performanceManifest string) (*performancev1alpha1.PerformanceProfile, error) {
	performanceScheme := runtime.NewScheme()
	apis.AddToScheme(performanceScheme)

	decode := serializer.NewCodecFactory(performanceScheme).UniversalDeserializer().Decode
	manifest, err := ioutil.ReadFile(performanceManifest)
	if err != nil {
		return nil, fmt.Errorf("Failed to read %s file", performanceManifest)
	}
	obj, _, err := decode([]byte(manifest), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to read the manifest file %s", performanceManifest)
	}
	profile, ok := obj.(*performancev1alpha1.PerformanceProfile)
	if !ok {
		return nil, fmt.Errorf("Failed to convert manifest file to profile")
	}
	return profile, nil
}

func testProfile() *performancev1alpha1.PerformanceProfile {
	reserved := performancev1alpha1.CPUSet("0")
	isolated := performancev1alpha1.CPUSet("1-3")
	hugePagesSize := performancev1alpha1.HugePageSize("1G")

	cnfRoleLabel := fmt.Sprintf("%s/%s", testutils.LabelRole, utils.RoleWorkerCNF)
	nodeSelector := map[string]string{cnfRoleLabel: ""}

	return &performancev1alpha1.PerformanceProfile{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PerformanceProfile",
			APIVersion: performancev1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: utils.PerformanceProfileName,
		},
		Spec: performancev1alpha1.PerformanceProfileSpec{
			CPU: &performancev1alpha1.CPU{
				Reserved: &reserved,
				Isolated: &isolated,
			},
			HugePages: &performancev1alpha1.HugePages{
				DefaultHugePagesSize: &hugePagesSize,
				Pages: []performancev1alpha1.HugePage{
					{
						Size:  "1G",
						Count: 1,
						Node:  pointer.Int32Ptr(0),
					},
				},
			},
			NodeSelector: nodeSelector,
			RealTimeKernel: &performancev1alpha1.RealTimeKernel{
				Enabled: pointer.BoolPtr(true),
			},
			AdditionalKernelArgs: []string{
				"nmi_watchdog=0",
				"audit=0",
				"mce=off",
				"processor.max_cstate=1",
				"idle=poll",
				"intel_idle.max_cstate=0",
			},
			NUMA: &performancev1alpha1.NUMA{
				TopologyPolicy: pointer.StringPtr("single-numa-node"),
			},
		},
	}
}
