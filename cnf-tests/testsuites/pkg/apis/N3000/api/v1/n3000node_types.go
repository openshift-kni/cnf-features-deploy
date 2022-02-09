// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2021 Intel Corporation

/*


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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// N3000NodeSpec defines the desired state of N3000Node
type N3000NodeSpec struct {
	// FPGA devices to be updated
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	FPGA []N3000Fpga `json:"fpga,omitempty"`
	// Fortville devices to be updated
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Fortville *N3000Fortville `json:"fortville,omitempty"`
	DryRun    bool            `json:"dryRun,omitempty"`
	// Allows for updating devices without draining the node
	DrainSkip bool `json:"drainSkip,omitempty"`
}

// N3000NodeStatus defines the observed state of N3000Node
type N3000NodeStatus struct {
	// Provides information about device update status
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Provides information about FPGA inventory on the node
	// +operator-sdk:csv:customresourcedefinitions:type=status
	FPGA []N3000FpgaStatus `json:"fpga,omitempty"`
	// Provides information about N3000 Fortville invetory on the node
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Fortville []N3000FortvilleStatus `json:"fortville,omitempty"`
}

type N3000FpgaStatus struct {
	PciAddr          string `json:"PCIAddr,omitempty"`
	DeviceID         string `json:"deviceId,omitempty"`
	BitstreamID      string `json:"bitstreamId,omitempty"`
	BitstreamVersion string `json:"bitstreamVersion,omitempty"`
	BootPage         string `json:"bootPage,omitempty"`
	NumaNode         int    `json:"numaNode,omitempty"`
}

type N3000FortvilleStatus struct {
	N3000PCI string            `json:"N3000PCI,omitempty"`
	NICs     []FortvilleStatus `json:"NICs,omitempty"`
}

type FortvilleStatus struct {
	Name    string `json:"name,omitempty"`
	PciAddr string `json:"PCIAddr,omitempty"`
	Version string `json:"NVMVersion,omitempty"`
	MAC     string `json:"MAC,omitempty"`
}

type N3000FortvilleStatusModules struct {
	Type    string `json:"type,omitempty"`
	Version string `json:"version,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Flash",type=string,JSONPath=`.status.conditions[?(@.type=="Flashed")].reason`

// N3000Node is the Schema for the n3000nodes API
// +operator-sdk:csv:customresourcedefinitions:displayName="N3000Node",resources={{N3000Node,v1,node}}
type N3000Node struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   N3000NodeSpec   `json:"spec,omitempty"`
	Status N3000NodeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// N3000NodeList contains a list of N3000Node
type N3000NodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []N3000Node `json:"items"`
}

func init() {
	SchemeBuilder.Register(&N3000Node{}, &N3000NodeList{})
}
