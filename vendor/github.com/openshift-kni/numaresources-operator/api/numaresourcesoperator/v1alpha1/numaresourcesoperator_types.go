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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1 "github.com/openshift/api/operator/v1"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
)

// NUMAResourcesOperatorSpec defines the desired state of NUMAResourcesOperator
type NUMAResourcesOperatorSpec struct {
	// Group of Nodes to enable RTE on
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Group of nodes to enable RTE on"
	NodeGroups []NodeGroup `json:"nodeGroups,omitempty"`
	// Optional Resource Topology Exporter image URL
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Optional RTE image URL",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	ExporterImage string `json:"imageSpec,omitempty"`
	// Valid values are: "Normal", "Debug", "Trace", "TraceAll".
	// Defaults to "Normal".
	// +optional
	// +kubebuilder:default=Normal
	LogLevel operatorv1.LogLevel `json:"logLevel,omitempty"`
}

// NodeGroup defines group of nodes that will run resource topology exporter daemon set
// You can choose the group of node by MachineConfigPoolSelector or by NodeSelector
type NodeGroup struct {
	// MachineConfigPoolSelector defines label selector for the machine config pool
	MachineConfigPoolSelector *metav1.LabelSelector `json:"machineConfigPoolSelector,omitempty"`
}

// NUMAResourcesOperatorStatus defines the observed state of NUMAResourcesOperator
type NUMAResourcesOperatorStatus struct {
	// DaemonSets of the configured RTEs, one per node group
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="RTE DaemonSets"
	DaemonSets []NamespacedName `json:"daemonsets,omitempty"`
	// MachineConfigPools resolved from configured node groups
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="RTE MCPs from node groups"
	MachineConfigPools []MachineConfigPool `json:"machineconfigpools,omitempty"`
	// Conditions show the current state of the NUMAResourcesOperator Operator
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// MachineConfigPool defines the observed state of each MachineConfigPool selected by node groups
type MachineConfigPool struct {
	// Name the name of the machine config pool
	Name string `json:"name"`

	// Conditions represents the latest available observations of MachineConfigPool current state.
	// +optional
	Conditions []mcov1.MachineConfigPoolCondition `json:"conditions,omitempty"`
}

//+genclient
//+genclient:nonNamespaced
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=numaresop,path=numaresourcesoperators,scope=Cluster

// NUMAResourcesOperator is the Schema for the numaresourcesoperators API
//+operator-sdk:csv:customresourcedefinitions:displayName="NUMA Resources Operator",resources={{DaemonSet,v1,rte-daemonset,ConfigMap,v1,rte-configmap}}
type NUMAResourcesOperator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NUMAResourcesOperatorSpec   `json:"spec,omitempty"`
	Status NUMAResourcesOperatorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NUMAResourcesOperatorList contains a list of NUMAResourcesOperator
type NUMAResourcesOperatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NUMAResourcesOperator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NUMAResourcesOperator{}, &NUMAResourcesOperatorList{})
}
