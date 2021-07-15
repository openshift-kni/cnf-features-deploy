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
	Interface   *string `json:"interface"`
	Ptp4lOpts   *string `json:"ptp4lOpts,omitempty"`
	Phc2sysOpts *string `json:"phc2sysOpts,omitempty"`
	Ptp4lConf   *string `json:"ptp4lConf,omitempty"`
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
