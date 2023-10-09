// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Subject defines the resource that can be used as PlacementBinding subject
type Subject struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Enum=policy.open-cluster-management.io
	APIGroup string `json:"apiGroup"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Enum=Policy;PolicySet
	Kind string `json:"kind"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// PlacementSubject defines the resource that can be used as PlacementBinding placementRef
type PlacementSubject struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Enum=apps.open-cluster-management.io;cluster.open-cluster-management.io
	APIGroup string `json:"apiGroup"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Enum=PlacementRule;Placement
	Kind string `json:"kind"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// PlacementBindingStatus defines the observed state of PlacementBinding
type PlacementBindingStatus struct{}

// BindingOverrides defines the overrides to the Subjects
type BindingOverrides struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=Enforce;enforce
	// This field overrides the policy remediationAction on target clusters
	RemediationAction string `json:"remediationAction,omitempty"`
}

// SubFilter defines the selection rule for bound clusters
type SubFilter string

const Restricted SubFilter = "restricted"

//+kubebuilder:object:root=true

// PlacementBinding is the Schema for the placementbindings API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=placementbindings,scope=Namespaced
// +kubebuilder:resource:path=placementbindings,shortName=pb
type PlacementBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +kubebuilder:validation:Optional
	BindingOverrides BindingOverrides `json:"bindingOverrides,omitempty"`
	// This field provides the ability to select a subset of bound clusters
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=restricted
	SubFilter SubFilter `json:"subFilter,omitempty"`
	// +kubebuilder:validation:Required
	PlacementRef PlacementSubject `json:"placementRef"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Subjects []Subject              `json:"subjects"`
	Status   PlacementBindingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PlacementBindingList contains a list of PlacementBinding
type PlacementBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PlacementBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PlacementBinding{}, &PlacementBindingList{})
}
