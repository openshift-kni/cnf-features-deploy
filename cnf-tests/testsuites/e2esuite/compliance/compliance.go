package compliance

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	compliancev1alpha1 "github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/execute"
)

var scanProfiles = []string{"ocp4-moderate", "ocp4-moderate-node", "rhcos4-moderate"}
var ignoreChecks = []string{"sysctl-net-ipv4-conf-all-log-martians", "sysctl-net-ipv4-conf-default-log-martians", "sysctl-net-ipv4-icmp-ignore-bogus-error-responses"}

var _ = Describe("[compliance]", func() {

	execute.BeforeAll(func() {
		profiles := []compliancev1alpha1.NamedObjectReference{}
		for _, profile := range scanProfiles {
			profiles = append(profiles, compliancev1alpha1.NamedObjectReference{
				Name:     profile,
				Kind:     "Profile",
				APIGroup: "compliance.openshift.io/v1alpha1",
			})
		}

		ssb := &compliancev1alpha1.ScanSettingBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "compliance-test-scan",
				Namespace: "openshift-compliance",
			},
			Profiles: profiles,
			SettingsRef: &compliancev1alpha1.NamedObjectReference{
				Name:     "default",
				Kind:     "ScanSetting",
				APIGroup: "compliance.openshift.io/v1alpha1",
			},
		}

		By("starting compliance scans")
		err := client.Client.Create(context.TODO(), ssb, &runtimeClient.CreateOptions{})
		if k8serrors.IsAlreadyExists(err) {
			csList := &compliancev1alpha1.ComplianceScanList{}
			err := client.Client.List(context.TODO(), csList)
			Expect(err).ToNot(HaveOccurred())

			for _, cs := range csList.Items {
				if cs.Annotations != nil {
					for _, profile := range scanProfiles {
						if strings.Contains(cs.Name, profile) {
							cs.Annotations["compliance.openshift.io/rescan"] = ""
							err = client.Client.Update(context.Background(), &cs)
							Expect(err).ToNot(HaveOccurred())
							break
						}
					}
				}
			}
		} else {
			Expect(err).ToNot(HaveOccurred())
		}

		By("waiting for compliance scans to finish")
		Eventually(func() bool {
			csList := &compliancev1alpha1.ComplianceScanList{}
			err := client.Client.List(context.TODO(), csList)
			Expect(err).ToNot(HaveOccurred())
			for _, cs := range csList.Items {
				for _, profile := range scanProfiles {
					if strings.Contains(cs.Name, profile) {
						if cs.Status.Phase != compliancev1alpha1.PhaseDone {
							return false
						}
						break
					}
				}
			}
			return true
		}, 20*time.Minute, 1*time.Minute).Should(BeTrue())
	})

	Context("validate compliance", func() {
		It("should check all compliance checks passed", func() {
			checkRes := &compliancev1alpha1.ComplianceCheckResultList{}
			err := client.Client.List(context.TODO(), checkRes)
			Expect(err).ToNot(HaveOccurred())

			failedScans := ""
			for _, res := range checkRes.Items {
				if res.Status == "FAIL" && (res.Severity == compliancev1alpha1.CheckResultSeverityHigh || res.Severity == compliancev1alpha1.CheckResultSeverityUnknown) {
					ignore := false
					for _, check := range ignoreChecks {
						if strings.Contains(res.Name, check) {
							ignore = true
							break
						}
					}
					if !ignore {
						failedScans += res.Name + "\n"
					}
				}
			}

			if failedScans != "" {
				Expect(fmt.Errorf("The following scans failed:\n" + failedScans)).ToNot(HaveOccurred())
			}
		})
	})
})
