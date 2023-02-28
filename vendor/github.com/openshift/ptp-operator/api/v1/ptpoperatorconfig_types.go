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

// PtpOperatorConfigSpec defines the desired state of PtpOperatorConfig
type PtpOperatorConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	DaemonNodeSelector map[string]string `json:"daemonNodeSelector"`
	// EventConfig to configure event sidecar
	EventConfig *PtpEventConfig `json:"ptpEventConfig,omitempty"`
}

// PtpOperatorConfigStatus defines the observed state of PtpOperatorConfig
type PtpOperatorConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Event Enabled",type="boolean",JSONPath=".spec.ptpEventConfig.enableEventPublisher",description="Event Enabled"

// PtpOperatorConfig is the Schema for the ptpoperatorconfigs API
type PtpOperatorConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PtpOperatorConfigSpec   `json:"spec,omitempty"`
	Status PtpOperatorConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PtpOperatorConfigList contains a list of PtpOperatorConfig
type PtpOperatorConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PtpOperatorConfig `json:"items"`
}

// PtpEventConfig defines the desired state of event framework
type PtpEventConfig struct {
	// +kubebuilder:default=false
	// EnableEventPublisher will deploy event proxy as a sidecar
	EnableEventPublisher bool `json:"enableEventPublisher,omitempty"`
	// TransportHost format is <protocol>://<transport-service>.<namespace>.svc.cluster.local:<transport-port>"
	// Example HTTP transport: "http://ptp-event-publisher-service.openshift-ptp.svc.cluster.local:9043"
	// Example AMQP transport: "amqp://amq-router-service-name.amq-namespace.svc.cluster.local"
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Transport Host",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	TransportHost string `json:"transportHost,omitempty"`
}

func init() {
	SchemeBuilder.Register(&PtpOperatorConfig{}, &PtpOperatorConfigList{})
}
