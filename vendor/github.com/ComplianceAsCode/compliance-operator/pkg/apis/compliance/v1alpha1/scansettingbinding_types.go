package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ScanSettingBindingStatusPhase string

const (
	ScanSettingBindingPhasePending   ScanSettingBindingStatusPhase = "PENDING"
	ScanSettingBindingPhaseReady     ScanSettingBindingStatusPhase = "READY"
	ScanSettingBindingPhaseInvalid   ScanSettingBindingStatusPhase = "INVALID"
	ScanSettingBindingPhaseSuspended ScanSettingBindingStatusPhase = "SUSPENDED"
)

type NamedObjectReference struct {
	Name     string `json:"name,omitempty"`
	Kind     string `json:"kind,omitempty"`
	APIGroup string `json:"apiGroup,omitempty"`
}

// +kubebuilder:object:root=true

// ScanSettingBinding is the Schema for the scansettingbindings API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=scansettingbindings,scope=Namespaced,shortName=ssb
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=`.status.phase`
type ScanSettingBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec     ScanSettingBindingSpec `json:"spec,omitempty"`
	Profiles []NamedObjectReference `json:"profiles,omitempty"`
	// +kubebuilder:default={"name":"default","kind": "ScanSetting", "apiGroup": "compliance.openshift.io/v1alpha1"}
	SettingsRef *NamedObjectReference `json:"settingsRef,omitempty"`
	// +optional
	Status ScanSettingBindingStatus `json:"status,omitempty"`
}

// This is a dummy spec to accommodate https://github.com/operator-framework/operator-sdk/issues/5584
type ScanSettingBindingSpec struct{}

type ScanSettingBindingStatus struct {
	Phase ScanSettingBindingStatusPhase `json:"phase,omitempty"`
	// +optional
	Conditions Conditions `json:"conditions,omitempty"`
	// Reference to the object generated from this ScanSettingBinding
	// +optional
	// +nullable
	OutputRef *corev1.TypedLocalObjectReference `json:"outputRef,omitempty"`
}

// +kubebuilder:object:root=true

// ScanSettingBindingList contains a list of ScanSettingBinding
type ScanSettingBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScanSettingBinding `json:"items"`
}

func (s *ScanSettingBindingStatus) SetConditionPending() {
	s.Conditions.SetCondition(Condition{
		Type:    "Ready",
		Status:  corev1.ConditionFalse,
		Reason:  "Pending",
		Message: "The scan setting binding is waiting to be processed",
	})
}

func (s *ScanSettingBindingStatus) SetConditionInvalid(msg string) {
	s.Conditions.SetCondition(Condition{
		Type:    "Ready",
		Status:  corev1.ConditionFalse,
		Reason:  "Invalid",
		Message: msg,
	})
}

func (s *ScanSettingBindingStatus) SetConditionReady() {
	s.Conditions.SetCondition(Condition{
		Type:    "Ready",
		Status:  corev1.ConditionTrue,
		Reason:  "Processed",
		Message: "The scan setting binding was successfully processed",
	})
}

func (s *ScanSettingBindingStatus) SetConditionSuspended() {
	s.Conditions.SetCondition(Condition{
		Type:    "Ready",
		Status:  corev1.ConditionFalse,
		Reason:  "Suspended",
		Message: "The scan setting binding uses a scan setting that is suspended",
	})
}

func init() {
	SchemeBuilder.Register(&ScanSettingBinding{}, &ScanSettingBindingList{})
}
