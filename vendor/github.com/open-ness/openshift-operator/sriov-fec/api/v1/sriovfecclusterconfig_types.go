// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2021 Intel Corporation

package v1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SyncStatus string

var (
	// InProgressSync indicates that the synchronization of the CR is in progress
	InProgressSync SyncStatus = "InProgress"
	// SucceededSync indicates that the synchronization of the CR succeeded
	SucceededSync SyncStatus = "Succeeded"
	// FailedSync indicates that the synchronization of the CR failed
	FailedSync SyncStatus = "Failed"
	// IgnoredSync indicates that the CR is ignored
	IgnoredSync SyncStatus = "Ignored"
)

func (udq *UplinkDownlinkQueues) String() string {
	return fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d,%d", udq.VF0, udq.VF1, udq.VF2, udq.VF3,
		udq.VF4, udq.VF5, udq.VF6, udq.VF7)
}

type UplinkDownlinkQueues struct {
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=32
	VF0 int `json:"vf0"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=32
	VF1 int `json:"vf1"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=32
	VF2 int `json:"vf2"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=32
	VF3 int `json:"vf3"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=32
	VF4 int `json:"vf4"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=32
	VF5 int `json:"vf5"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=32
	VF6 int `json:"vf6"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=32
	VF7 int `json:"vf7"`
}

type UplinkDownlink struct {
	// +kubebuilder:validation:Minimum=0
	Bandwidth int `json:"bandwidth"`
	// +kubebuilder:validation:Minimum=0
	LoadBalance int                  `json:"loadBalance"`
	Queues      UplinkDownlinkQueues `json:"queues"`
}

// N3000BBDevConfig specifies variables to configure N3000 with
type N3000BBDevConfig struct {
	// +kubebuilder:validation:Enum=FPGA_5GNR;FPGA_LTE
	NetworkType string `json:"networkType"`
	PFMode      bool   `json:"pfMode"`
	// +kubebuilder:validation:Minimum=0
	FLRTimeOut int            `json:"flrTimeout"`
	Downlink   UplinkDownlink `json:"downlink"`
	Uplink     UplinkDownlink `json:"uplink"`
}

type QueueGroupConfig struct {
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=8
	NumQueueGroups int `json:"numQueueGroups"`
	// +kubebuilder:validation:Minimum=16
	// +kubebuilder:validation:Maximum=16
	NumAqsPerGroups int `json:"numAqsPerGroups"`
	// +kubebuilder:validation:Minimum=4
	// +kubebuilder:validation:Maximum=4
	AqDepthLog2 int `json:"aqDepthLog2"`
}

// ACC100BBDevConfig specifies variables to configure ACC100 with
type ACC100BBDevConfig struct {
	PFMode bool `json:"pfMode"`
	// +kubebuilder:validation:Minimum=16
	// +kubebuilder:validation:Maximum=16
	NumVfBundles int `json:"numVfBundles"`
	// +kubebuilder:validation:Minimum=1024
	// +kubebuilder:validation:Maximum=1024
	MaxQueueSize int              `json:"maxQueueSize"`
	Uplink4G     QueueGroupConfig `json:"uplink4G"`
	Downlink4G   QueueGroupConfig `json:"downlink4G"`
	Uplink5G     QueueGroupConfig `json:"uplink5G"`
	Downlink5G   QueueGroupConfig `json:"downlink5G"`
}

// BBDevConfig is a struct containing configuration for various FEC cards
type BBDevConfig struct {
	N3000  *N3000BBDevConfig  `json:"n3000,omitempty"`
	ACC100 *ACC100BBDevConfig `json:"acc100,omitempty"`
}

// PhysicalFunctionConfig defines a possible configuration of a single Physical Function (PF), i.e. card
type PhysicalFunctionConfig struct {
	// PCIAdress is a Physical Functions's PCI address that will be configured according to this spec
	// +kubebuilder:validation:Pattern=`^[a-fA-F0-9]{4}:[a-fA-F0-9]{2}:[01][a-fA-F0-9]\.[0-7]$`
	PCIAddress string `json:"pciAddress"`
	// PFDriver to bound the PFs to
	PFDriver string `json:"pfDriver"`
	// VFDriver to bound the VFs to
	VFDriver string `json:"vfDriver"`
	// VFAmount is an amount of VFs to be created
	// +kubebuilder:validation:Minimum=0
	VFAmount int `json:"vfAmount"`
	// BBDevConfig is a config for PF's queues
	BBDevConfig BBDevConfig `json:"bbDevConfig"`
}

type NodeConfig struct {
	// Name of the node
	NodeName string `json:"nodeName"`
	// List of physical functions (cards) configs
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PhysicalFunctions []PhysicalFunctionConfig `json:"physicalFunctions"`
}

// SriovFecClusterConfigSpec defines the desired state of SriovFecClusterConfig
type SriovFecClusterConfigSpec struct {
	// List of node configurations
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Nodes     []NodeConfig `json:"nodes"`
	DrainSkip bool         `json:"drainSkip,omitempty"`
}

// SriovFecClusterConfigStatus defines the observed state of SriovFecClusterConfig
type SriovFecClusterConfigStatus struct {
	// Indicates the synchronization status of the CR
	// +operator-sdk:csv:customresourcedefinitions:type=status
	SyncStatus    SyncStatus `json:"syncStatus,omitempty"`
	LastSyncError string     `json:"lastSyncError,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="SyncStatus",type=string,JSONPath=`.status.syncStatus`

// SriovFecClusterConfig is the Schema for the sriovfecclusterconfigs API
// +operator-sdk:csv:customresourcedefinitions:displayName="SriovFecClusterConfig",resources={{SriovFecNodeConfig,v1,node}}
type SriovFecClusterConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SriovFecClusterConfigSpec   `json:"spec,omitempty"`
	Status SriovFecClusterConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SriovFecClusterConfigList contains a list of SriovFecClusterConfig
type SriovFecClusterConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SriovFecClusterConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SriovFecClusterConfig{}, &SriovFecClusterConfigList{})
}
