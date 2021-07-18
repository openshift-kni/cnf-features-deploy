package empty_string

import (
	"context"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	clientmachineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	testclient "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/execute"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/images"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/pods"
)

type empty struct{}
type semaphore chan empty

// acquire n resources
func (s semaphore) P(n int) {
	e := empty{}
	for i := 0; i < n; i++ {
		s <- e
	}
}

// release n resources
func (s semaphore) V(n int) {
	for i := 0; i < n; i++ {
		<-s
	}
}

/* mutexes */

func (s semaphore) Lock() {
	s.P(1)
}

func (s semaphore) Unlock() {
	s.V(1)
}

/* signal-wait */

func (s semaphore) Signal() {
	s.V(1)
}

func (s semaphore) Wait(n int) {
	s.P(n)
}

var (
	machineConfigPoolNodeSelector string
)

func init() {
}

var _ = Describe("flaketests", func() {
	execute.BeforeAll(func() {
		var err error
		err = namespaces.Create(namespaces.FlakesTest, client.Client)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("general", func() {
		It("should report all machine config pools are in ready status", func() {
			mcp := &clientmachineconfigv1.MachineConfigPoolList{}
			err := testclient.Client.List(context.TODO(), mcp)
			Expect(err).ToNot(HaveOccurred())

			for _, mcItem := range mcp.Items {
				Expect(mcItem.Status.MachineCount).To(Equal(mcItem.Status.ReadyMachineCount))
			}
		})

		sem := make(semaphore, 100)
		for i := 0; i < 100; i++ {
			go func(i int) {
				It("should return the echo string", func() {
					By("Create a pod")
					pod := simplePod("empty_strings-" + strconv.Itoa(i))
					simplePod, err := client.Client.Pods(namespaces.FlakesTest).Create(context.Background(), pod, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					By("Check if the pod is running")
					Eventually(func() corev1.PodPhase {
						runningPod, err := client.Client.Pods(namespaces.FlakesTest).Get(context.Background(), simplePod.Name, metav1.GetOptions{})

						Expect(err).ToNot(HaveOccurred())
						return runningPod.Status.Phase
					}, 1*time.Minute, 1*time.Second).Should(Equal(corev1.PodRunning))

					By("comparing Echo")
					cmd := []string{"echo", "test"}
					out, err := pods.ExecCommand(client.Client, *simplePod, cmd)
					Expect(out.String()).Should(ContainSubstring("test"))
				})
				sem.Signal()
			}(i)
		}
		sem.Wait(100)

	})
})

func simplePod(name string) *corev1.Pod {
	res := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"name": name,
			},
			Namespace: namespaces.FlakesTest,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:    name,
					Image:   images.For(images.TestUtils),
					Command: []string{"/bin/sh", "-c"},
					Args:    []string{"sleep inf"},
				},
			},
		},
	}

	return &res
}
