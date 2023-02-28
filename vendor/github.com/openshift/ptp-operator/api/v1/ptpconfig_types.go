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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PtpConfigSpec defines the desired state of PtpConfig
type PtpConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Profile   []PtpProfile   `json:"profile"`
	Recommend []PtpRecommend `json:"recommend"`
}

// PtpConfigStatus defines the observed state of PtpConfig
type PtpConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	MatchList []NodeMatchList `json:"matchList,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PtpConfig is the Schema for the ptpconfigs API
type PtpConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PtpConfigSpec   `json:"spec,omitempty"`
	Status PtpConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PtpConfigList contains a list of PtpConfig
type PtpConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PtpConfig `json:"items"`
}

type PtpProfile struct {
	Name        *string `json:"name"`
	Interface   *string `json:"interface,omitempty"`
	Ptp4lOpts   *string `json:"ptp4lOpts,omitempty"`
	Phc2sysOpts *string `json:"phc2sysOpts,omitempty"`
	Ts2PhcOpts  *string `json:"ts2phcOpts,omitempty"`
	Ptp4lConf   *string `json:"ptp4lConf,omitempty"`
	Phc2sysConf *string `json:"phc2sysConf,omitempty"`
	Ts2PhcConf  *string `json:"ts2phcConf,omitempty"`
	// +kubebuilder:validation:Enum=SCHED_OTHER;SCHED_FIFO;
	PtpSchedulingPolicy *string `json:"ptpSchedulingPolicy,omitempty"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65
	PtpSchedulingPriority *int64             `json:"ptpSchedulingPriority,omitempty"`
	PtpClockThreshold     *PtpClockThreshold `json:"ptpClockThreshold,omitempty"`
	PtpSettings           map[string]string  `json:"ptpSettings,omitempty"`
}

type PtpClockThreshold struct {
	// +kubebuilder:default=5
	// clock state to stay in holdover state in secs
	HoldOverTimeout int64 `json:"holdOverTimeout,omitempty"`
	// +kubebuilder:default=100
	// max offset in nano secs
	MaxOffsetThreshold int64 `json:"maxOffsetThreshold,omitempty"`
	// +kubebuilder:default=-100
	// min offset in nano secs
	MinOffsetThreshold int64 `json:"minOffsetThreshold,omitempty"`
}

type PtpRecommend struct {
	Profile  *string     `json:"profile"`
	Priority *int64      `json:"priority"`
	Match    []MatchRule `json:"match,omitempty"`
}

type MatchRule struct {
	NodeLabel *string `json:"nodeLabel,omitempty"`
	NodeName  *string `json:"nodeName,omitempty"`
}

type NodeMatchList struct {
	NodeName *string `json:"nodeName"`
	Profile  *string `json:"profile"`
}

func init() {
	SchemeBuilder.Register(&PtpConfig{}, &PtpConfigList{})
}
