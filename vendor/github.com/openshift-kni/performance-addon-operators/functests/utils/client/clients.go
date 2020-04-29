package client

import (
	"context"
	"time"

	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	configv1 "github.com/openshift/api/config/v1"
	tunedv1 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	"github.com/openshift-kni/performance-addon-operators/pkg/apis"
)

var (
	// Client defines the API client to run CRUD operations, that will be used for testing
	Client client.Client
	// K8sClient defines k8s client to run subresource operations, for example you should use it to get pod logs
	K8sClient *kubernetes.Clientset
	// ClientsEnabled tells if the client from the package can be used
	ClientsEnabled bool
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

	var err error
	Client, err = New()
	if err != nil {
		klog.Info("Failed to initialize client, check the KUBECONFIG env variable", err.Error())
		ClientsEnabled = false
		return
	}
	K8sClient, err = NewK8s()
	if err != nil {
		klog.Info("Failed to initialize k8s client, check the KUBECONFIG env variable", err.Error())
		ClientsEnabled = false
		return
	}
	ClientsEnabled = true
}

// New returns a controller-runtime client.
func New() (client.Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	c, err := client.New(cfg, client.Options{})
	return c, err
}

// NewK8s returns a kubernetes clientset
func NewK8s() (*kubernetes.Clientset, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Exit(err.Error())
	}
	return clientset, nil
}

func GetWithRetry(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	var err error
	EventuallyWithOffset(1, func() bool {
		err = Client.Get(context.TODO(), key, obj)
		retry := doRetry(err)
		if retry {
			klog.Infof("Getting %s failed, retrying: %v", key.Name, err)
		} else if err != nil {
			klog.Infof("Getting %s failed, not retrying: %v", key.Name, err)
		}
		return retry
	}, 1*time.Minute, 10*time.Second).Should(BeFalse(), "Max numbers of retries reached")
	return err
}

func doRetry(err error) bool {
	if errors.IsServiceUnavailable(err) ||
		errors.IsTimeout(err) ||
		errors.IsServerTimeout(err) ||
		errors.IsTooManyRequests(err) ||
		errors.IsInternalError(err) ||
		errors.IsUnexpectedServerError(err) {
		return true
	}
	return false
}
