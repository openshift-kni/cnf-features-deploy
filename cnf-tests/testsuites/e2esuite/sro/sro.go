package sro

import (
	ctx "context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	srov1beta1 "github.com/openshift-psap/special-resource-operator/api/v1beta1"
	helmerv1beta1 "github.com/openshift-psap/special-resource-operator/pkg/helmer/api/v1beta1"
	ocpbuildv1 "github.com/openshift/api/build/v1"
	ocpimagev1 "github.com/openshift/api/image/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/pointer"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/execute"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/images"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/nodes"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/pods"
)

const (
	sourceImageStreamName      = "oot-source-driver"
	sourceImageBuildConfigName = "oot-source-driver-build"
	ootDriverImageName         = "ice-driver-container"
	chartName                  = "charts"
	chartsPath                 = "/usr/src/oot-driver/charts"
	indexFileName              = "index.yaml"
	iceDrivTarName             = "ice-driver-0.0.1.tgz"
	roleName                   = "chart-role"
	serviceAccountName         = "chart-sa"
	roleBindingName            = "chart-binding"
	specialResourceName        = "ice-driver"
)

var (
	driverToolKitImage = ""
)

type buildArgs struct {
	Name  string
	Value string
}

var _ = Describe("sro", func() {
	imageStreamValidation := false
	configMapValidation := false

	Context("Build source out of tree driver for SRO using", func() {
		BeforeEach(func() {
			sourceImageStream := &ocpimagev1.ImageStream{ObjectMeta: metav1.ObjectMeta{Name: sourceImageStreamName, Namespace: namespaces.SroTestNamespace}}
			err := client.Client.Create(ctx.TODO(), sourceImageStream)
			Expect(err).ToNot(HaveOccurred())

			bc, err := createBuildConfig()
			Expect(err).ToNot(HaveOccurred())

			build, err := startBuild(bc)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() ocpbuildv1.BuildPhase {
				err := client.Client.Get(ctx.TODO(), goclient.ObjectKey{Name: build.Name, Namespace: namespaces.SroTestNamespace}, build)
				Expect(err).ToNot(HaveOccurred())
				return build.Status.Phase
			}, 5*time.Minute, 5*time.Second).Should(Equal(ocpbuildv1.BuildPhaseComplete))
		})

		It("Should have the source driver image as imageStream", func() {
			sourceImageStream := &ocpimagev1.ImageStream{}
			err := client.Client.Get(ctx.TODO(), goclient.ObjectKey{Name: sourceImageStreamName, Namespace: namespaces.SroTestNamespace}, sourceImageStream)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(sourceImageStream.Status.Tags)).To(Equal(1))
			Expect(sourceImageStream.Status.Tags[0].Tag).To(Equal("latest"))
			imageStreamValidation = true
		})
	})

	Context("Apply the chart as a configmap for SRO to use", func() {
		BeforeEach(func() {
			// Create a role,sa and binding to allow a pod to create configmaps
			err := createRole()
			Expect(err).ToNot(HaveOccurred())
			err = createServiceAccount()
			Expect(err).ToNot(HaveOccurred())
			err = createRoleBinding()
			Expect(err).ToNot(HaveOccurred())

			// Start a cnf-test pod
			pod := pods.DefinePod(namespaces.SroTestNamespace)
			pod.Spec.ServiceAccountName = serviceAccountName
			pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{Privileged: pointer.BoolPtr(true)}
			pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
				{
					Name:      "kubelet",
					MountPath: "/kubelet",
					ReadOnly:  true,
				},
				{
					Name:      "ca",
					MountPath: "/etc/pki/ca-trust/",
					ReadOnly:  true,
				},
			}

			pod.Spec.Volumes = []corev1.Volume{
				{Name: "kubelet", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/var/lib/kubelet"}}},
				{Name: "ca", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/etc/pki/ca-trust/"}}},
			}

			err = client.Client.Create(ctx.TODO(), pod)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() corev1.PodPhase {
				err := client.Client.Get(ctx.TODO(), goclient.ObjectKey{Name: pod.Name, Namespace: namespaces.SroTestNamespace}, pod)
				Expect(err).ToNot(HaveOccurred())
				return pod.Status.Phase
			}, 2*time.Minute, 5*time.Second).Should(Equal(corev1.PodRunning))

			command := []string{"oc",
				"-n",
				namespaces.SroTestNamespace,
				"create",
				"cm",
				chartName,
				fmt.Sprintf("--from-file=%s/%s", chartsPath, indexFileName),
				fmt.Sprintf("--from-file=%s/%s", chartsPath, iceDrivTarName)}
			output, err := pods.ExecCommand(client.Client, *pod, command)
			Expect(err).ToNot(HaveOccurred(), output.String())

			command = []string{"oc",
				"adm",
				"release",
				"info",
				"-a",
				"/kubelet/config.json",
				"--image-for=driver-toolkit"}
			output, err = pods.ExecCommand(client.Client, *pod, command)
			Expect(err).ToNot(HaveOccurred(), output.String())
			driverToolKitImage = strings.TrimSuffix(output.String(), "\r\n")
		})

		It("should exist in the configmap object", func() {
			cm, err := client.Client.ConfigMaps(namespaces.SroTestNamespace).Get(ctx.TODO(), chartName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(cm.Data)).To(Equal(1))
			Expect(len(cm.BinaryData)).To(Equal(1))
			configMapValidation = true
		})
	})

	Context("Apply the special resource CR to build the OOT driver", func() {
		var containerTag string
		var imageStreamExist bool

		execute.BeforeAll(func() {
			Expect(configMapValidation).To(BeTrue())
			Expect(imageStreamValidation).To(BeTrue())

			sourceImageStream := &ocpimagev1.ImageStream{ObjectMeta: metav1.ObjectMeta{Name: ootDriverImageName, Namespace: namespaces.SroTestNamespace}}
			err := client.Client.Create(ctx.TODO(), sourceImageStream)
			Expect(err).ToNot(HaveOccurred())

			_, containerTag, err = createSpecialResource()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should have a oot driver imagestream built", func() {
			sourceImageStream := &ocpimagev1.ImageStream{}
			Eventually(func() bool {
				err := client.Client.Get(ctx.TODO(), goclient.ObjectKey{Name: ootDriverImageName, Namespace: namespaces.SroTestNamespace}, sourceImageStream)
				Expect(err).ToNot(HaveOccurred())
				return len(sourceImageStream.Status.Tags) == 1
			}, 10*time.Minute, 5*time.Second).Should(BeTrue())
			imageStreamExist = true
		})

		It("should have the driver built inside the container", func() {
			Expect(imageStreamExist).To(BeTrue())

			pod := pods.DefinePod(namespaces.SroTestNamespace)
			pod.Spec.Containers[0].Image = fmt.Sprintf("image-registry.openshift-image-registry.svc:5000/%s/%s:%s", namespaces.SroTestNamespace, ootDriverImageName, containerTag)

			err := client.Client.Create(ctx.TODO(), pod)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() corev1.PodPhase {
				err := client.Client.Get(ctx.TODO(), goclient.ObjectKey{Name: pod.Name, Namespace: namespaces.SroTestNamespace}, pod)
				Expect(err).ToNot(HaveOccurred())
				return pod.Status.Phase
			}, 2*time.Minute, 5*time.Second).Should(Equal(corev1.PodRunning))

			output, err := pods.ExecCommand(client.Client, *pod, []string{"ls", "/oot-driver/"})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.String()).To(ContainSubstring("ice.ko"))
		})
	})
})

func createBuildConfig() (*ocpbuildv1.BuildConfig, error) {
	dockerFile := fmt.Sprintf("FROM %s", images.For(images.TestUtils))
	output := &corev1.ObjectReference{
		Kind:      "ImageStreamTag",
		Namespace: namespaces.SroTestNamespace,
		Name:      fmt.Sprintf("%s:latest", sourceImageStreamName),
	}

	bc := &ocpbuildv1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{Name: sourceImageBuildConfigName, Namespace: namespaces.SroTestNamespace},
		Spec: ocpbuildv1.BuildConfigSpec{
			CommonSpec: ocpbuildv1.CommonSpec{
				Source: ocpbuildv1.BuildSource{Dockerfile: &dockerFile},
				Output: ocpbuildv1.BuildOutput{
					To: output,
				},
				Strategy: ocpbuildv1.BuildStrategy{Type: ocpbuildv1.DockerBuildStrategyType},
			},
		}}

	bc, err := client.Client.BuildConfigs(namespaces.SroTestNamespace).Create(ctx.TODO(), bc, metav1.CreateOptions{})

	return bc, err
}

func startBuild(bc *ocpbuildv1.BuildConfig) (*ocpbuildv1.Build, error) {
	buildRequest := &ocpbuildv1.BuildRequest{ObjectMeta: metav1.ObjectMeta{Name: bc.Name, Namespace: namespaces.SroTestNamespace}}
	return client.Client.BuildConfigs(namespaces.SroTestNamespace).Instantiate(ctx.TODO(), bc.Name, buildRequest, metav1.CreateOptions{})
}
func createRole() error {
	role := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: roleName, Namespace: namespaces.SroTestNamespace},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"*"},
			},
		}}

	err := client.Client.Create(ctx.TODO(), role)
	if err != nil {
		return err
	}

	clusterRole := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: roleName},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"config.openshift.io"},
				Resources: []string{"clusterversions"},
				Verbs:     []string{"get"},
			},
		}}

	err = client.Client.Create(ctx.TODO(), clusterRole)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func createServiceAccount() error {
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: serviceAccountName, Namespace: namespaces.SroTestNamespace}}

	return client.Client.Create(ctx.TODO(), sa)
}

func createRoleBinding() error {
	roleBinding := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: roleBindingName, Namespace: namespaces.SroTestNamespace},
		RoleRef: rbacv1.RoleRef{
			Name:     roleName,
			Kind:     "Role",
			APIGroup: "rbac.authorization.k8s.io",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: serviceAccountName,
			},
		},
	}

	err := client.Client.Create(ctx.TODO(), roleBinding)
	if err != nil {
		return err
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: roleBindingName},
		RoleRef: rbacv1.RoleRef{
			Name:     roleName,
			Kind:     "ClusterRole",
			APIGroup: "rbac.authorization.k8s.io",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccountName,
				Namespace: namespaces.SroTestNamespace,
			},
		},
	}

	err = client.Client.Create(ctx.TODO(), clusterRoleBinding)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func createSpecialResource() (*srov1beta1.SpecialResource, string, error) {
	nodesObjList := &corev1.NodeList{}
	err := client.Client.List(ctx.TODO(), nodesObjList)
	if err != nil {
		return nil, "", err
	}
	nodesList := make([]string, len(nodesObjList.Items))
	for idx, node := range nodesObjList.Items {
		nodesList[idx] = node.Name
	}

	nn, err := nodes.MatchingOptionalSelectorByName(nodesList)
	if err != nil {
		return nil, "", err
	}

	if len(nn) == 0 {
		return nil, "", fmt.Errorf("0 nodes match the selector")
	}

	kernels := make(map[string]bool)
	kernel := ""
	for _, nodeName := range nn {
		node, err := client.Client.Nodes().Get(ctx.TODO(), nodeName, metav1.GetOptions{})
		if err != nil {
			return nil, "", err
		}

		kernels[node.Status.NodeInfo.KernelVersion] = true
		kernel = node.Status.NodeInfo.KernelVersion
	}

	if len(kernels) == 0 {
		return nil, "", fmt.Errorf("unable to find kernel version")
	}

	sp := &srov1beta1.SpecialResource{
		ObjectMeta: metav1.ObjectMeta{Name: specialResourceName, Namespace: namespaces.SroTestNamespace},
		Spec: srov1beta1.SpecialResourceSpec{
			Namespace: namespaces.SroTestNamespace,
			Chart: helmerv1beta1.HelmChart{
				Name:    specialResourceName,
				Version: "0.0.1",
				Repository: helmerv1beta1.HelmRepo{
					Name: "chart",
					URL:  fmt.Sprintf("cm://%s/%s", namespaces.SroTestNamespace, chartName),
				},
			},
			Set: unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":             "Values",
					"apiVersion":       "sro.openshift.io/v1beta1",
					"kmodNames":        []string{"ice"},
					"containerName":    ootDriverImageName,
					"externalRegistry": "",
					"signDriver":       false,
					"downloadDriver":   true,
					"kernelVersion":    kernel,
					"buildArgs": []buildArgs{
						{Name: "KMODVER", Value: "SRO"},
						{Name: "IMAGE", Value: driverToolKitImage},
						{Name: "OUTPUT_IMAGE", Value: driverToolKitImage},
						{Name: "KERNEL_SOURCE", Value: "yum"},
						{Name: "ICE_DRIVER_VERSION", Value: "1.7.16"},
					},
				},
			},
			DriverContainer: srov1beta1.SpecialResourceDriverContainer{
				Artifacts: srov1beta1.SpecialResourceArtifacts{
					Images: []srov1beta1.SpecialResourceImages{
						{
							Name:      fmt.Sprintf("%s:latest", sourceImageStreamName),
							Kind:      "ImageStreamTag",
							Namespace: namespaces.SroTestNamespace,
							Paths: []srov1beta1.SpecialResourcePaths{
								{
									SourcePath:     "/usr/src/oot-driver/.",
									DestinationDir: "./",
								},
							},
						},
					},
				},
			},
		},
	}

	err = client.Client.Create(ctx.TODO(), sp)

	return sp, kernel, err
}

// This will remove all the cluster scope level objects
func Clean() {
	specialResourceList := &srov1beta1.SpecialResourceList{}
	err := client.Client.List(ctx.TODO(), specialResourceList)
	if meta.IsNoMatchError(err) {
		return
	}
	Expect(err).ToNot(HaveOccurred())

	for _, specialResource := range specialResourceList.Items {
		if specialResource.Name == "special-resource-preamble" {
			continue
		}

		err = client.Client.Delete(ctx.TODO(), &specialResource)
		Expect(err).ToNot(HaveOccurred())
	}

	_, err = client.Client.ClusterRoles().Get(ctx.TODO(), roleName, metav1.GetOptions{})
	if err == nil {
		err = client.Client.ClusterRoles().Delete(ctx.TODO(), roleName, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	_, err = client.Client.ClusterRoleBindings().Get(ctx.TODO(), roleBindingName, metav1.GetOptions{})
	if err == nil {
		err = client.Client.ClusterRoleBindings().Delete(ctx.TODO(), roleBindingName, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(err).ToNot(HaveOccurred())
	}
}
