// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2021 Intel Corporation

package v2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VF struct {
	PCIAddress string `json:"pciAddress"`
	Driver     string `json:"driver"`
	DeviceID   string `json:"deviceID"`
}

type SriovAccelerator struct {
	VendorID   string `json:"vendorID"`
	DeviceID   string `json:"deviceID"`
	PCIAddress string `json:"pciAddress"`
	PFDriver   string `json:"driver"`
	MaxVFs     int    `json:"maxVirtualFunctions"`
	VFs        []VF   `json:"virtualFunctions"`
}

type NodeInventory struct {
	SriovAccelerators []SriovAccelerator `json:"sriovAccelerators,omitempty"`
}

// SriovFecNodeConfigSpec defines the desired state of SriovFecNodeConfig
type SriovFecNodeConfigSpec struct {
	// List of PhysicalFunctions configs
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PhysicalFunctions []PhysicalFunctionConfigExt `json:"physicalFunctions"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// Skips drain process when true; default false. Should be true if operator is running on SNO
	DrainSkip bool `json:"drainSkip,omitempty"`
}

// SriovFecNodeConfigStatus defines the observed state of SriovFecNodeConfig
type SriovFecNodeConfigStatus struct {
	// Provides information about device update status
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Provides information about FPGA inventory on the node
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Inventory NodeInventory `json:"inventory,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Configured",type=string,JSONPath=`.status.conditions[?(@.type=="Configured")].reason`
// +kubebuilder:storageversion
// +kubebuilder:resource:shortName=sfnc

// SriovFecNodeConfig is the Schema for the sriovfecnodeconfigs API
// +operator-sdk:csv:customresourcedefinitions:displayName="SriovFecNodeConfig",resources={{SriovFecNodeConfig,v1,node}}
type SriovFecNodeConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SriovFecNodeConfigSpec   `json:"spec,omitempty"`
	Status SriovFecNodeConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SriovFecNodeConfigList contains a list of SriovFecNodeConfig
type SriovFecNodeConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SriovFecNodeConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SriovFecNodeConfig{}, &SriovFecNodeConfigList{})
}
