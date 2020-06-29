package client

import (
	"os"

	perfApi "github.com/openshift-kni/performance-addon-operators/pkg/apis"
	configv1 "github.com/openshift/api/config/v1"
	clientconfigv1 "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	tunedv1 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	clientmachineconfigv1 "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/typed/machineconfiguration.openshift.io/v1"
	ptpv1 "github.com/openshift/ptp-operator/pkg/client/clientset/versioned/typed/ptp/v1"
	sriovk8sv1 "github.com/openshift/sriov-network-operator/pkg/apis/k8s/v1"
	sriovv1 "github.com/openshift/sriov-network-operator/pkg/apis/sriovnetwork/v1"

	"github.com/golang/glog"
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
)

// Client defines the client set that will be used for testing
var Client *ClientSet

func init() {
	Client = New("")
}

// ClientSet provides the struct to talk with relevant API
type ClientSet struct {
	client.Client
	corev1client.CoreV1Interface
	clientconfigv1.ConfigV1Interface
	clientmachineconfigv1.MachineconfigurationV1Interface
	networkv1client.NetworkingV1Client
	appsv1client.AppsV1Interface
	discovery.DiscoveryInterface
	ptpv1.PtpV1Interface
	Config *rest.Config
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
		glog.Infof("Failed to init kubernetes client, please check the $KUBECONFIG environment variable: %s", err)
		return nil
	}

	clientSet := &ClientSet{}
	clientSet.CoreV1Interface = corev1client.NewForConfigOrDie(config)
	clientSet.ConfigV1Interface = clientconfigv1.NewForConfigOrDie(config)
	clientSet.MachineconfigurationV1Interface = clientmachineconfigv1.NewForConfigOrDie(config)
	clientSet.AppsV1Interface = appsv1client.NewForConfigOrDie(config)
	clientSet.DiscoveryInterface = discovery.NewDiscoveryClientForConfigOrDie(config)
	clientSet.NetworkingV1Client = *networkv1client.NewForConfigOrDie(config)
	clientSet.PtpV1Interface = ptpv1.NewForConfigOrDie(config)
	clientSet.Config = config

	myScheme := runtime.NewScheme()
	if err = scheme.AddToScheme(myScheme); err != nil {
		panic(err)
	}

	// Setup Scheme for all resources
	if err := perfApi.AddToScheme(myScheme); err != nil {
		panic(err)
	}

	if err := configv1.AddToScheme(myScheme); err != nil {
		panic(err)
	}

	if err := mcov1.AddToScheme(myScheme); err != nil {
		panic(err)
	}

	if err := tunedv1.AddToScheme(myScheme); err != nil {
		panic(err)
	}

	if err := sriovk8sv1.SchemeBuilder.AddToScheme(myScheme); err != nil {
		panic(err)
	}

	if err := sriovv1.AddToScheme(myScheme); err != nil {
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
