package client

import (
	"os"

	gkopv1alpha "github.com/gatekeeper/gatekeeper-operator/api/v1alpha1"
	sriovk8sv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	sriovv1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	fpgav1 "github.com/open-ness/openshift-operator/N3000/api/v1"
	fecv1 "github.com/open-ness/openshift-operator/sriov-fec/api/v1"
	gkv1alpha "github.com/open-policy-agent/gatekeeper/apis/mutations/v1alpha1"
	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	srov1beta1 "github.com/openshift-psap/special-resource-operator/api/v1beta1"
	ocpbuildv1 "github.com/openshift/api/build/v1"
	configv1 "github.com/openshift/api/config/v1"
	ocpv1 "github.com/openshift/api/config/v1"
	clientconfigv1 "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	imagev1client "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	tunedv1 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	clientmachineconfigv1 "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/typed/machineconfiguration.openshift.io/v1"
	ptpv1 "github.com/openshift/ptp-operator/pkg/client/clientset/versioned/typed/ptp/v1"

	"github.com/golang/glog"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	discovery "k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/scheme"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	networkv1client "k8s.io/client-go/kubernetes/typed/networking/v1"
	rbacv1client "k8s.io/client-go/kubernetes/typed/rbac/v1"
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
	rbacv1client.RbacV1Interface
	discovery.DiscoveryInterface
	ptpv1.PtpV1Interface
	imagev1client.ImageV1Interface
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
		glog.Infof("Failed to init kubernetes client, please check the $KUBECONFIG environment variable")
		return nil
	}

	clientSet := &ClientSet{}
	clientSet.CoreV1Interface = corev1client.NewForConfigOrDie(config)
	clientSet.ConfigV1Interface = clientconfigv1.NewForConfigOrDie(config)
	clientSet.MachineconfigurationV1Interface = clientmachineconfigv1.NewForConfigOrDie(config)
	clientSet.AppsV1Interface = appsv1client.NewForConfigOrDie(config)
	clientSet.RbacV1Interface = rbacv1client.NewForConfigOrDie(config)
	clientSet.DiscoveryInterface = discovery.NewDiscoveryClientForConfigOrDie(config)
	clientSet.NetworkingV1Client = *networkv1client.NewForConfigOrDie(config)
	clientSet.PtpV1Interface = ptpv1.NewForConfigOrDie(config)
	clientSet.ImageV1Interface = imagev1client.NewForConfigOrDie(config)
	clientSet.Config = config

	myScheme := runtime.NewScheme()
	if err = scheme.AddToScheme(myScheme); err != nil {
		panic(err)
	}

	// Setup Scheme for all resources
	if err := performancev2.AddToScheme(myScheme); err != nil {
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

	if err := gkv1alpha.AddToScheme(myScheme); err != nil {
		panic(err)
	}

	if err := fpgav1.AddToScheme(myScheme); err != nil {
		panic(err)
	}

	if err := fecv1.AddToScheme(myScheme); err != nil {
		panic(err)
	}

	if err := gkopv1alpha.AddToScheme(myScheme); err != nil {
		panic(err)
	}

	if err := nfdv1.AddToScheme(myScheme); err != nil {
		panic(err)
	}

	if err := srov1beta1.AddToScheme(myScheme); err != nil {
		panic(err)
	}

	if err := ocpv1.Install(myScheme); err != nil {
		panic(err)
	}

	if err := ocpbuildv1.Install(myScheme); err != nil {
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
