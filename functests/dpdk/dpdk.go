package dpdk

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-kni/cnf-features-deploy/functests/utils/client"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/execute"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/nodes"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/pods"
)

const (
	dpdkHostRole         = "worker"
	hostnameLabel         = "kubernetes.io/hostname"
	dpdkAnnotationNetwork = "dpdk-network"
	testDpdkNamespace     = "dpdk-testing"
	testCmdPath           = "/opt/test.sh"

)

var dpdkAppImage string

func init() {
	// Set DPDK app image
	dpdkAppImage = os.Getenv("DPDK_APP_IMAGE")
	if dpdkAppImage == "" {
		dpdkAppImage = "quay.io/schseba/dpdk-s2i-base:ds"
	}
}

var _ = Describe("dpdk", func() {
	var nList []corev1.Node

	execute.BeforeAll(func() {
		// create the namespace
		err := namespaces.Create(testDpdkNamespace, client.Client)
		Expect(err).ToNot(HaveOccurred())

		// clean up the namespace
		err = namespaces.Clean(testDpdkNamespace, client.Client)
		Expect(err).ToNot(HaveOccurred())

		// get nodes
		nList, err = nodes.GetByRole(client.Client, dpdkHostRole)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(nList)).ShouldNot(Equal(0))
	})

	var _ = Context("Run a sanity test on each worker", func() {
		var configMapName string
		var p *corev1.Pod
		AfterEach(func() {
			pods.WaitForDeletion(client.Client,p, 2*time.Minute)
			deleteTestpmdConfigMap(configMapName)
		})
		It("Should forward and receive packets", func() {
			c := createTestpmdConfigMap(testDpdkNamespace)
			configMapName = c.Name
			for _, n := range nList {
				p = createTestPod(n.Name, testDpdkNamespace, c.Name)
				pods.WaitForCondition(client.Client, p,corev1.ContainersReady, corev1.ConditionTrue, 2*time.Minute)
				By(fmt.Sprintf("Executing %s inside the pod %s", testCmdPath, p.Name))
				out, err := exec.Command("oc", "rsh", "-n", p.Namespace, p.Name, "bash", testCmdPath).CombinedOutput()
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("cannot execute %s inside the pod %s", testCmdPath, p.Name))
				By("Parsing output from the DPDK application")
				checkRxTx(string(out))
			}
		})
	})
})

// creteTestPod creates a pod that will act as a runtime for the DPDK test application
func createTestPod(nodeName, namespace, configMapName string) *corev1.Pod {
	defaultMode := int32(0755)

	res := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-dpdk",
			Labels: map[string]string{
				"app": "test-dpdk",
			},
			Annotations: map[string]string{
				"k8s.v1.cni.cncf.io/networks": dpdkAnnotationNetwork,
			},
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:  "test-dpdk",
					Image: dpdkAppImage,
					SecurityContext: &corev1.SecurityContext{
						Capabilities: &corev1.Capabilities{
							Add: []corev1.Capability{"IPC_LOCK"},
						},
					},
					Command:         []string{"/bin/bash", "-c", "--"},
					Args:            []string{"while true; do sleep inf; done;"},
					ImagePullPolicy: "Always",
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:                     resource.MustParse("4"),
							corev1.ResourceMemory:                  resource.MustParse("1000Mi"),
							corev1.ResourceHugePagesPrefix + "1Gi": resource.MustParse("4Gi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:                     resource.MustParse("4"),
							corev1.ResourceMemory:                  resource.MustParse("1000Mi"),
							corev1.ResourceHugePagesPrefix + "1Gi": resource.MustParse("4Gi"),
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "hugepage",
							MountPath: "/mnt/huge",
							ReadOnly:  false,
						},
						{
							Name:      "testcmd",
							MountPath: testCmdPath,
							SubPath:   "test.sh",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "hugepage",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{
							Medium: corev1.StorageMediumHugePages,
						},
					},
				},
				{
					Name: "testcmd",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							corev1.LocalObjectReference{Name: configMapName},
							nil,
							&defaultMode,
							nil,
						},
					},
				},
			},

			NodeSelector: map[string]string{
				hostnameLabel: nodeName,
			},
		},
	}

	By("Creating a test pod")
	p, err := client.Client.Pods(namespace).Create(res)
	Expect(err).ToNot(HaveOccurred(), "cannot create the test pod " + p.Name)
	return p
}

// createTestpmdConfigMap creates a ConfigMap that mounts testpmd wrapper script
// The script is a mix of bash and expect. Expect is required to interact with
// the testpmd application
func createTestpmdConfigMap(namespace string) *corev1.ConfigMap {
	m := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testpmd",
			Namespace: namespace,
		},
		Data: map[string]string{
			"test.sh": `
#!/usr/bin/bash
CPU=$(cat /sys/fs/cgroup/cpuset/cpuset.cpus)
echo ${CPU}
echo ${PCIDEVICE_OPENSHIFT_IO_DPDKNIC}

TMPEXPECT=$(mktemp /tmp/expect.XXXXXX)

/bin/cat<<-EOL > $TMPEXPECT
  spawn testpmd -l ${CPU} -w ${PCIDEVICE_OPENSHIFT_IO_DPDKNIC}  -- -i --portmask=0x1 --nb-cores=2 --forward-mode=mac --port-topology=loop
  set timeout 10000
  expect "testpmd>"
  send -- "start\r"
  sleep 10
  expect "testpmd>"
  send -- "stop\r"
  expect "testpmd>"
  send -- "quit\r"
  expect eof
EOL

chmod 700 $TMPEXPECT
expect -f $TMPEXPECT
                       `,
		},
	}

	By("Create testpmd wrapper script")
	m, err := client.Client.ConfigMaps(testDpdkNamespace).Create(m)
	Expect(err).ToNot(HaveOccurred(), "cannot create testpmd wrapper script")
	return m
}

// deleteTestpmdConfigMap 
func deleteTestpmdConfigMap(configMapName string) {
	By("Deleting configMap " + configMapName)
	err := client.Client.ConfigMaps(testDpdkNamespace).Delete(configMapName, &metav1.DeleteOptions{})
	Expect(err).ToNot(HaveOccurred())
}

// checkRxTx parses the output from the DPDK test application
// and verifies that packets have passed the NIC TX and RX queues
func checkRxTx(out string) {
	str := strings.Split(out, "\n")
	for i := 0; i < len(str); i++ {
		if strings.Contains(str[i], "all ports") {
			i++
			d := getNumberOfPackets(str, i)
			Expect(d).Should(BeNumerically(">", 0), "number of received packets should be greater than 0")

			i++
	        d = getNumberOfPackets(str, i)
			Expect(d).Should(BeNumerically(">", 0), "number of transferred packets should be greater than 0")
		}
	}
}

// getNumber of packets parses the string (represented as a slice)
// and returns an element representing the number of packets
func getNumberOfPackets(s []string, index int) int {
	r := strings.Fields(s[index])
	Expect(len(r)).To(Equal(6), "the slice doesn't contain 6 elements")
	d, err := strconv.Atoi(r[1])
	Expect(err).ToNot(HaveOccurred())
    return d
}

