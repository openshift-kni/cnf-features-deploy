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

package fake

import (
	"context"
	json "encoding/json"
	"fmt"

	v1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	storagev1 "k8s.io/client-go/applyconfigurations/storage/v1"
	testing "k8s.io/client-go/testing"
)

// FakeCSINodes implements CSINodeInterface
type FakeCSINodes struct {
	Fake *FakeStorageV1
}

var csinodesResource = v1.SchemeGroupVersion.WithResource("csinodes")

var csinodesKind = v1.SchemeGroupVersion.WithKind("CSINode")

// Get takes name of the cSINode, and returns the corresponding cSINode object, and an error if there is any.
func (c *FakeCSINodes) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.CSINode, err error) {
	emptyResult := &v1.CSINode{}
	obj, err := c.Fake.
		Invokes(testing.NewRootGetActionWithOptions(csinodesResource, name, options), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.CSINode), err
}

// List takes label and field selectors, and returns the list of CSINodes that match those selectors.
func (c *FakeCSINodes) List(ctx context.Context, opts metav1.ListOptions) (result *v1.CSINodeList, err error) {
	emptyResult := &v1.CSINodeList{}
	obj, err := c.Fake.
		Invokes(testing.NewRootListActionWithOptions(csinodesResource, csinodesKind, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.CSINodeList{ListMeta: obj.(*v1.CSINodeList).ListMeta}
	for _, item := range obj.(*v1.CSINodeList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested cSINodes.
func (c *FakeCSINodes) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchActionWithOptions(csinodesResource, opts))
}

// Create takes the representation of a cSINode and creates it.  Returns the server's representation of the cSINode, and an error, if there is any.
func (c *FakeCSINodes) Create(ctx context.Context, cSINode *v1.CSINode, opts metav1.CreateOptions) (result *v1.CSINode, err error) {
	emptyResult := &v1.CSINode{}
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateActionWithOptions(csinodesResource, cSINode, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.CSINode), err
}

// Update takes the representation of a cSINode and updates it. Returns the server's representation of the cSINode, and an error, if there is any.
func (c *FakeCSINodes) Update(ctx context.Context, cSINode *v1.CSINode, opts metav1.UpdateOptions) (result *v1.CSINode, err error) {
	emptyResult := &v1.CSINode{}
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateActionWithOptions(csinodesResource, cSINode, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.CSINode), err
}

// Delete takes name of the cSINode and deletes it. Returns an error if one occurs.
func (c *FakeCSINodes) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(csinodesResource, name, opts), &v1.CSINode{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeCSINodes) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	action := testing.NewRootDeleteCollectionActionWithOptions(csinodesResource, opts, listOpts)

	_, err := c.Fake.Invokes(action, &v1.CSINodeList{})
	return err
}

// Patch applies the patch and returns the patched cSINode.
func (c *FakeCSINodes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.CSINode, err error) {
	emptyResult := &v1.CSINode{}
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceActionWithOptions(csinodesResource, name, pt, data, opts, subresources...), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.CSINode), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied cSINode.
func (c *FakeCSINodes) Apply(ctx context.Context, cSINode *storagev1.CSINodeApplyConfiguration, opts metav1.ApplyOptions) (result *v1.CSINode, err error) {
	if cSINode == nil {
		return nil, fmt.Errorf("cSINode provided to Apply must not be nil")
	}
	data, err := json.Marshal(cSINode)
	if err != nil {
		return nil, err
	}
	name := cSINode.Name
	if name == nil {
		return nil, fmt.Errorf("cSINode.Name must be provided to Apply")
	}
	emptyResult := &v1.CSINode{}
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceActionWithOptions(csinodesResource, *name, types.ApplyPatchType, data, opts.ToPatchOptions()), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.CSINode), err
}
