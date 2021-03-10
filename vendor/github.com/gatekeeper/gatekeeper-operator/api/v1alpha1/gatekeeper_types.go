/*


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
	admregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
// Important: Run "make" to regenerate code after modifying this file

// GatekeeperSpec defines the desired state of Gatekeeper
type GatekeeperSpec struct {
	// +optional
	Image *ImageConfig `json:"image,omitempty"`
	// +optional
	Audit *AuditConfig `json:"audit,omitempty"`
	// +optional
	ValidatingWebhook *WebhookMode `json:"validatingWebhook,omitempty"`
	// +optional
	MutatingWebhook *WebhookMode `json:"mutatingWebhook,omitempty"`
	// +optional
	Webhook *WebhookConfig `json:"webhook,omitempty"`
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// +optional
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
}

type ImageConfig struct {
	// Image to pull including registry (optional), repository, name, and tag
	// e.g. quay.io/gatekeeper/gatekeeper-operator:latest
	// +optional
	Image *string `json:"image,omitempty"`
	// +optional
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

type AuditConfig struct {
	// +kubebuilder:validation:Minimum:=0
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// +optional
	AuditInterval *metav1.Duration `json:"auditInterval,omitempty"`
	// +kubebuilder:validation:Minimum:=0
	// +optional
	ConstraintViolationLimit *uint64 `json:"constraintViolationLimit,omitempty"`
	// +optional
	AuditFromCache *AuditFromCacheMode `json:"auditFromCache,omitempty"`
	// +kubebuilder:validation:Minimum:=0
	// +optional
	AuditChunkSize *uint64 `json:"auditChunkSize,omitempty"`
	// +optional
	LogLevel *LogLevelMode `json:"logLevel,omitempty"`
	// +optional
	EmitAuditEvents *EmitEventsMode `json:"emitAuditEvents,omitempty"`
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// +kubebuilder:validation:Enum:=Enabled;Disabled
type WebhookMode string

const (
	WebhookEnabled  WebhookMode = "Enabled"
	WebhookDisabled WebhookMode = "Disabled"
)

type WebhookConfig struct {
	// +kubebuilder:validation:Minimum:=0
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// +optional
	LogLevel *LogLevelMode `json:"logLevel,omitempty"`
	// +optional
	EmitAdmissionEvents *EmitEventsMode `json:"emitAdmissionEvents,omitempty"`
	// +optional
	FailurePolicy *admregv1.FailurePolicyType `json:"failurePolicy,omitempty"`
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// +kubebuilder:validation:Enum:=DEBUG;INFO;WARNING;ERROR
type LogLevelMode string

const (
	LogLevelDEBUG   LogLevelMode = "DEBUG"
	LogLevelInfo    LogLevelMode = "INFO"
	LogLevelWarning LogLevelMode = "WARNING"
	LogLevelError   LogLevelMode = "ERROR"
)

// +kubebuilder:validation:Enum:=Enabled;Disabled
type AuditFromCacheMode string

const (
	AuditFromCacheEnabled  AuditFromCacheMode = "Enabled"
	AuditFromCacheDisabled AuditFromCacheMode = "Disabled"
)

// +kubebuilder:validation:Enum:=Enabled;Disabled
type EmitEventsMode string

const (
	EmitEventsEnabled  EmitEventsMode = "Enabled"
	EmitEventsDisabled EmitEventsMode = "Disabled"
)

// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
// Important: Run "make" to regenerate code after modifying this file

// GatekeeperStatus defines the observed state of Gatekeeper
type GatekeeperStatus struct {
	// ObservedGeneration is the generation as observed by the operator consuming this API.
	ObservedGeneration int64             `json:"observedGeneration"`
	AuditConditions    []StatusCondition `json:"auditConditions"`
	WebhookConditions  []StatusCondition `json:"webhookConditions"`
}

// StatusCondition describes the current state of a component.
type StatusCondition struct {
	// Type of status condition.
	Type StatusConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// Last time the condition was checked.
	// +optional
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// Last time the condition transit from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// (brief) reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:validation:Enum:=Ready;Not Ready
type StatusConditionType string

const (
	StatusReady    StatusConditionType = "Ready"
	StatusNotReady StatusConditionType = "Not Ready"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=gatekeepers,scope=Cluster
//// +kubebuilder:printcolumn:name="Audit Status",type=string,JSONPath=`.status.auditConditions[0].type`,description="The status of the Gatekeeper Audit"
//// +kubebuilder:printcolumn:name="Webhook Status",type=string,JSONPath=`.status.webhookConditions[0].type`,description="The status of the Gatekeeper Webhook"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Gatekeeper is the Schema for the gatekeepers API
type Gatekeeper struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GatekeeperSpec   `json:"spec,omitempty"`
	Status GatekeeperStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GatekeeperList contains a list of Gatekeeper
type GatekeeperList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Gatekeeper `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Gatekeeper{}, &GatekeeperList{})
}
