/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"time"

	v1 "github.com/openshift/ptp-operator/pkg/apis/ptp/v1"
	scheme "github.com/openshift/ptp-operator/pkg/client/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// PtpConfigsGetter has a method to return a PtpConfigInterface.
// A group's client should implement this interface.
type PtpConfigsGetter interface {
	PtpConfigs(namespace string) PtpConfigInterface
}

// PtpConfigInterface has methods to work with PtpConfig resources.
type PtpConfigInterface interface {
	Create(*v1.PtpConfig) (*v1.PtpConfig, error)
	Update(*v1.PtpConfig) (*v1.PtpConfig, error)
	UpdateStatus(*v1.PtpConfig) (*v1.PtpConfig, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error
	Get(name string, options metav1.GetOptions) (*v1.PtpConfig, error)
	List(opts metav1.ListOptions) (*v1.PtpConfigList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.PtpConfig, err error)
	PtpConfigExpansion
}

// ptpConfigs implements PtpConfigInterface
type ptpConfigs struct {
	client rest.Interface
	ns     string
}

// newPtpConfigs returns a PtpConfigs
func newPtpConfigs(c *PtpV1Client, namespace string) *ptpConfigs {
	return &ptpConfigs{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the ptpConfig, and returns the corresponding ptpConfig object, and an error if there is any.
func (c *ptpConfigs) Get(name string, options metav1.GetOptions) (result *v1.PtpConfig, err error) {
	result = &v1.PtpConfig{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("ptpconfigs").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of PtpConfigs that match those selectors.
func (c *ptpConfigs) List(opts metav1.ListOptions) (result *v1.PtpConfigList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.PtpConfigList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("ptpconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested ptpConfigs.
func (c *ptpConfigs) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("ptpconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a ptpConfig and creates it.  Returns the server's representation of the ptpConfig, and an error, if there is any.
func (c *ptpConfigs) Create(ptpConfig *v1.PtpConfig) (result *v1.PtpConfig, err error) {
	result = &v1.PtpConfig{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("ptpconfigs").
		Body(ptpConfig).
		Do().
		Into(result)
	return
}

// Update takes the representation of a ptpConfig and updates it. Returns the server's representation of the ptpConfig, and an error, if there is any.
func (c *ptpConfigs) Update(ptpConfig *v1.PtpConfig) (result *v1.PtpConfig, err error) {
	result = &v1.PtpConfig{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("ptpconfigs").
		Name(ptpConfig.Name).
		Body(ptpConfig).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *ptpConfigs) UpdateStatus(ptpConfig *v1.PtpConfig) (result *v1.PtpConfig, err error) {
	result = &v1.PtpConfig{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("ptpconfigs").
		Name(ptpConfig.Name).
		SubResource("status").
		Body(ptpConfig).
		Do().
		Into(result)
	return
}

// Delete takes name of the ptpConfig and deletes it. Returns an error if one occurs.
func (c *ptpConfigs) Delete(name string, options *metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("ptpconfigs").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *ptpConfigs) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("ptpconfigs").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched ptpConfig.
func (c *ptpConfigs) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.PtpConfig, err error) {
	result = &v1.PtpConfig{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("ptpconfigs").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
