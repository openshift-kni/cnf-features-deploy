package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NodePtpDeviceSpec defines the desired state of NodePtpDevice
// +k8s:openapi-gen=true
type NodePtpDeviceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

type PtpDevice struct {
	Name	string	`json:"name,omitempty"`
	Profile	string	`json:"profile,omitempty"`
}

// NodePtpDeviceStatus defines the observed state of NodePtpDevice
// +k8s:openapi-gen=true
type NodePtpDeviceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Devices	[]PtpDevice	`json:"devices,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodePtpDevice is the Schema for the nodeptpdevices API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type NodePtpDevice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodePtpDeviceSpec   `json:"spec,omitempty"`
	Status NodePtpDeviceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodePtpDeviceList contains a list of NodePtpDevice
type NodePtpDeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodePtpDevice `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodePtpDevice{}, &NodePtpDeviceList{})
}
