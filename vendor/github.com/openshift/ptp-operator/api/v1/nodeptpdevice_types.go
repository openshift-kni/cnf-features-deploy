/*
Copyright 2021.

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

package v1

import (
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NodePtpDeviceSpec defines the desired state of NodePtpDevice
type NodePtpDeviceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

}

type PtpDevice struct {
	Name    string `json:"name,omitempty"`
	Profile string `json:"profile,omitempty"`
}

// NodePtpDeviceStatus defines the observed state of NodePtpDevice
type NodePtpDeviceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Devices  []PtpDevice `json:"devices,omitempty"`
	Hwconfig []HwConfig  `json:"hwconfig,omitempty"`
}

type HwConfig struct {
	DeviceID string              `json:"deviceID,omitempty"`
	VendorID string              `json:"vendorID,omitempty"`
	Failed   bool                `json:"failed,omitempty"`
	Status   string              `json:"status,omitempty"`
	Config   *apiextensions.JSON `json:"config,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NodePtpDevice is the Schema for the nodeptpdevices API
type NodePtpDevice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodePtpDeviceSpec   `json:"spec,omitempty"`
	Status NodePtpDeviceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NodePtpDeviceList contains a list of NodePtpDevice
type NodePtpDeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodePtpDevice `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodePtpDevice{}, &NodePtpDeviceList{})
}
