package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PtpConfigSpec defines the desired state of PtpConfig
// +k8s:openapi-gen=true
type PtpConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Profile		[]PtpProfile	`json:"profile"`
	Recommend	[]PtpRecommend	`json:"recommend"`
}

type PtpProfile struct {
	Name		*string	`json:"name"`
	Interface	*string	`json:"interface"`
	Ptp4lOpts	*string	`json:"ptp4lOpts,omitempty"`
	Phc2sysOpts	*string	`json:"phc2sysOpts,omitempty"`
	Ptp4lConf	*string	`json:"ptp4lConf,omitempty"`
}

type PtpRecommend struct {
	Profile		*string		`json:"profile"`
	Priority	*int64		`json:"priority"`
	Match		[]MatchRule	`json:"match,omitempty"`
}

type MatchRule struct {
	NodeLabel	*string	`json:"nodeLabel,omitempty"`
	NodeName	*string	`json:"nodeName,omitempty"`
}

// PtpConfigStatus defines the observed state of PtpConfig
// +k8s:openapi-gen=true
type PtpConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	MatchList	[]NodeMatchList	`json:"matchList,omitempty"`
}

type NodeMatchList struct {
	NodeName	*string	`json:"nodeName"`
	Profile		*string	`json:"profile"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PtpConfig is the Schema for the ptpconfigs API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type PtpConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PtpConfigSpec   `json:"spec,omitempty"`
	Status PtpConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PtpConfigList contains a list of PtpConfig
type PtpConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PtpConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PtpConfig{}, &PtpConfigList{})
}
