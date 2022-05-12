/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clients

import (
	"context"
	"time"

	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	nrtv1alpha1 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha1"
	numaresourcesoperatorv1alpha1 "github.com/openshift-kni/numaresources-operator/api/numaresourcesoperator/v1alpha1"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
	if err := mcov1.AddToScheme(scheme.Scheme); err != nil {
		klog.Exit(err.Error())
	}

	if err := numaresourcesoperatorv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		klog.Exit(err.Error())
	}

	if err := apiextensionsv1.AddToScheme(scheme.Scheme); err != nil {
		klog.Exit(err.Error())
	}

	if err := nrtv1alpha1.AddToScheme(scheme.Scheme); err != nil {
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

func GetWithRetry(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	var err error
	EventuallyWithOffset(1, func() error {
		err = Client.Get(ctx, key, obj)
		if err != nil {
			klog.Infof("Getting %s failed, retrying: %v", key.Name, err)
		}
		return err
	}, 1*time.Minute, 10*time.Second).ShouldNot(HaveOccurred(), "Max numbers of retries getting %v reached", key)
	return err
}
