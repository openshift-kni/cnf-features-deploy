package client

import (
	"github.com/openshift-kni/performance-addon-operators/pkg/apis"
	configv1 "github.com/openshift/api/config/v1"
	tunedv1 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	// Client defines the API client to run CRUD operations, that will be used for testing
	Client client.Client
	// K8sClient defines k8s client to run subresource operations, for example you should use it to get pod logs
	K8sClient *kubernetes.Clientset
)

func init() {
	// Setup Scheme for all resources
	if err := apis.AddToScheme(scheme.Scheme); err != nil {
		klog.Exit(err.Error())
	}

	if err := configv1.AddToScheme(scheme.Scheme); err != nil {
		klog.Exit(err.Error())
	}

	if err := mcov1.AddToScheme(scheme.Scheme); err != nil {
		klog.Exit(err.Error())
	}

	if err := tunedv1.AddToScheme(scheme.Scheme); err != nil {
		klog.Exit(err.Error())
	}

	Client = New()
	K8sClient = NewK8s()
}

// New returns a controller-runtime client.
func New() client.Client {
	cfg, err := config.GetConfig()
	if err != nil {
		klog.Exit(err.Error())
	}

	c, err := client.New(cfg, client.Options{})
	if err != nil {
		klog.Exit(err.Error())
	}

	return c
}

// NewK8s returns a kubernetes clientset
func NewK8s() *kubernetes.Clientset {
	cfg, err := config.GetConfig()
	if err != nil {
		klog.Exit(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Exit(err.Error())
	}
	return clientset
}
