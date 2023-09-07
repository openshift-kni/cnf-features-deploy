package client

import (
	"os"

	"github.com/golang/glog"

	clientconfigv1 "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	prometheusClientV1 "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	discovery "k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/scheme"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	networkv1client "k8s.io/client-go/kubernetes/typed/networking/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ptpv1api "github.com/openshift/ptp-operator/api/v1"
	ptpv1fake "github.com/openshift/ptp-operator/pkg/client/clientset/versioned/fake"
	ptpv1 "github.com/openshift/ptp-operator/pkg/client/clientset/versioned/typed/ptp/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	k8sFakeClient "k8s.io/client-go/kubernetes/fake"
)

// Client defines the client set that will be used for testing
var Client = &ClientSet{}

// ClientSet provides the struct to talk with relevant API
type ClientSet struct {
	client.Client
	kubernetes.Interface
	networkv1client.NetworkingV1Client
	appsv1client.AppsV1Interface
	discovery.DiscoveryInterface
	ptpv1.PtpV1Interface
	Config    *rest.Config
	OcpClient clientconfigv1.ConfigV1Interface
	corev1client.CoreV1Interface
	KubeConfigPath string
	prometheusClientV1.MonitoringV1Client
}

func Setup() {
	Client = New("")
}

// New returns a *ClientBuilder with the given kubeconfig.
func New(kubeconfig string) *ClientSet {
	var config *rest.Config
	var err error

	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}

	if kubeconfig != "" {
		glog.V(4).Infof("Loading kube client config from path %q", kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		glog.V(4).Infof("Using in-cluster kube client config")
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		glog.Infof("Failed to create a valid client,  environement variable.")
		// Cannot create client, nothing else to do
		os.Exit(1)
	}

	clientSet := &ClientSet{}
	// Save the kubeconfig for later use
	clientSet.KubeConfigPath = kubeconfig
	clientSet.CoreV1Interface = corev1client.NewForConfigOrDie(config)
	clientSet.Interface = kubernetes.NewForConfigOrDie(config)
	clientSet.AppsV1Interface = appsv1client.NewForConfigOrDie(config)
	clientSet.DiscoveryInterface = discovery.NewDiscoveryClientForConfigOrDie(config)
	clientSet.NetworkingV1Client = *networkv1client.NewForConfigOrDie(config)
	clientSet.PtpV1Interface = ptpv1.NewForConfigOrDie(config)
	clientSet.OcpClient = clientconfigv1.NewForConfigOrDie(config)
	clientSet.MonitoringV1Client = *prometheusClientV1.NewForConfigOrDie(config)
	clientSet.Config = config

	myScheme := runtime.NewScheme()
	if err = scheme.AddToScheme(myScheme); err != nil {
		panic(err)
	}

	if err := apiext.AddToScheme(myScheme); err != nil {
		panic(err)
	}

	clientSet.Client, err = client.New(config, client.Options{
		Scheme: myScheme,
	})

	if err != nil {
		return nil
	}

	return clientSet
}

// GetTestClientSet Overwrites the existing clientholders with a mocked version for unit testing.
func GetTestClientSet(k8sMockObjects []runtime.Object) *ClientSet {
	// Build slices of different objects depending on what client
	// is supposed to expect them.
	var ptpClientObjects []runtime.Object
	var k8sClientObjects []runtime.Object

	for _, v := range k8sMockObjects {
		// Based on what type of object is, populate certain object slices
		// with what is supported by a certain client.
		// Add more items below if/when needed.
		switch v.(type) {
		// K8s Client Objects
		case *ptpv1api.PtpConfig:
			ptpClientObjects = append(ptpClientObjects, v)
		case *corev1.Node:
			k8sClientObjects = append(k8sClientObjects, v)
		}

	}

	// Add the objects to their corresponding API Clients
	Client.Interface = k8sFakeClient.NewSimpleClientset(k8sClientObjects...)
	Client.PtpV1Interface = ptpv1fake.NewSimpleClientset(ptpClientObjects...).PtpV1()

	return Client
}

func ClearTestClientsHolder() {
	if Client != nil {
		Client.Interface = nil
		Client.PtpV1Interface = nil
	}
}
