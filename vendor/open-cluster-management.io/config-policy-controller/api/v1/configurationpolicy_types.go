// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package v1

import (
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// A custom type is required since there is no way to have a kubebuilder marker
// apply to the items of a slice.

// +kubebuilder:validation:MinLength=1
type NonEmptyString string

// RemediationAction : enforce or inform
// +kubebuilder:validation:Enum=Inform;inform;Enforce;enforce
type RemediationAction string

// Severity : low, medium, high, or critical
// +kubebuilder:validation:Enum=low;Low;medium;Medium;high;High;critical;Critical
type Severity string

// PruneObjectBehavior is used to remove objects that are managed by the
// policy upon policy deletion.
// +kubebuilder:validation:Enum=DeleteAll;DeleteIfCreated;None;
type PruneObjectBehavior string

const (
	// Enforce is an remediationAction to make changes
	Enforce RemediationAction = "Enforce"

	// Inform is an remediationAction to only inform
	Inform RemediationAction = "Inform"
)

// ComplianceState shows the state of enforcement
type ComplianceState string

const (
	// Compliant is an ComplianceState
	Compliant ComplianceState = "Compliant"

	// NonCompliant is an ComplianceState
	NonCompliant ComplianceState = "NonCompliant"

	// UnknownCompliancy is an ComplianceState
	UnknownCompliancy ComplianceState = "UnknownCompliancy"

	// Terminating is a ComplianceState
	Terminating ComplianceState = "Terminating"
)

// Condition is the base struct for representing resource conditions
type Condition struct {
	// Type of condition, e.g Complete or Failed.
	Type string `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status,omitempty" protobuf:"bytes,12,rep,name=status"`
	// The last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,3,opt,name=lastTransitionTime"`
	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,4,opt,name=reason"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,5,opt,name=message"`
}

type Target struct {
	// 'include' is an array of filepath expressions to include objects by name.
	Include []NonEmptyString `json:"include,omitempty"`
	// 'exclude' is an array of filepath expressions to exclude objects by name.
	Exclude []NonEmptyString `json:"exclude,omitempty"`
	// 'matchLabels' is a map of {key,value} pairs matching objects by label.
	MatchLabels *map[string]string `json:"matchLabels,omitempty"`
	// 'matchExpressions' is an array of label selector requirements matching objects by label.
	MatchExpressions *[]metav1.LabelSelectorRequirement `json:"matchExpressions,omitempty"`
}

// Define String() so that the LabelSelector is dereferenced in the logs
func (t Target) String() string {
	fmtSelectorStr := "{include:%s,exclude:%s,matchLabels:%+v,matchExpressions:%+v}"
	if t.MatchLabels == nil && t.MatchExpressions == nil {
		return fmt.Sprintf(fmtSelectorStr, t.Include, t.Exclude, nil, nil)
	}

	if t.MatchLabels == nil {
		return fmt.Sprintf(fmtSelectorStr, t.Include, t.Exclude, nil, *t.MatchExpressions)
	}

	if t.MatchExpressions == nil {
		return fmt.Sprintf(fmtSelectorStr, t.Include, t.Exclude, *t.MatchLabels, nil)
	}

	return fmt.Sprintf(fmtSelectorStr, t.Include, t.Exclude, *t.MatchLabels, *t.MatchExpressions)
}

// Configures the minimum elapsed time before a ConfigurationPolicy is reevaluated
type EvaluationInterval struct {
	//+kubebuilder:validation:Pattern=`^(?:(?:(?:[0-9]+(?:.[0-9])?)(?:h|m|s|(?:ms)|(?:us)|(?:ns)))|never)+$`
	// The minimum elapsed time before a ConfigurationPolicy is reevaluated when in the compliant state. Set this to
	// "never" to disable reevaluation when in the compliant state.
	Compliant string `json:"compliant,omitempty"`
	//+kubebuilder:validation:Pattern=`^(?:(?:(?:[0-9]+(?:.[0-9])?)(?:h|m|s|(?:ms)|(?:us)|(?:ns)))|never)+$`
	// The minimum elapsed time before a ConfigurationPolicy is reevaluated when in the noncompliant state. Set this to
	// "never" to disable reevaluation when in the noncompliant state.
	NonCompliant string `json:"noncompliant,omitempty"`
}

var ErrIsNever = errors.New("the interval is set to never")

// parseInterval converts the input string to a duration. The default value is 0s. ErrIsNever is returned when the
// string is set to "never".
func (e EvaluationInterval) parseInterval(interval string) (time.Duration, error) {
	if interval == "" {
		return 0, nil
	}

	if interval == "never" {
		return 0, ErrIsNever
	}

	parsedInterval, err := time.ParseDuration(interval)
	if err != nil {
		return 0, err
	}

	return parsedInterval, nil
}

// GetCompliantInterval converts the Compliant interval to a duration. ErrIsNever is returned when the string
// is set to "never".
func (e EvaluationInterval) GetCompliantInterval() (time.Duration, error) {
	return e.parseInterval(e.Compliant)
}

// GetNonCompliantInterval converts the NonCompliant interval to a duration. ErrIsNever is returned when the string
// is set to "never".
func (e EvaluationInterval) GetNonCompliantInterval() (time.Duration, error) {
	return e.parseInterval(e.NonCompliant)
}

// ConfigurationPolicySpec defines the desired state of ConfigurationPolicy
type ConfigurationPolicySpec struct {
	Severity          Severity          `json:"severity,omitempty"`          // low, medium, high
	RemediationAction RemediationAction `json:"remediationAction,omitempty"` // enforce, inform
	// 'namespaceSelector' defines the list of namespaces to include/exclude for objects defined in
	// spec.objectTemplates. All selector rules are ANDed. If 'include' is not provided but
	// 'matchLabels' and/or 'matchExpressions' are, 'include' will behave as if ['*'] were given. If
	// 'matchExpressions' and 'matchLabels' are both not provided, 'include' must be provided to
	// retrieve namespaces.
	NamespaceSelector Target `json:"namespaceSelector,omitempty"`
	// 'object-templates' and 'object-templates-raw' are arrays of objects for the configuration
	// policy to check, create, modify, or delete on the cluster. 'object-templates' is an array
	// of objects, while 'object-templates-raw' is a string containing an array of objects in
	// YAML format. Only one of the two object-templates variables can be set in a given
	// configurationPolicy.
	ObjectTemplates []*ObjectTemplate `json:"object-templates,omitempty"`
	// 'object-templates' and 'object-templates-raw' are arrays of objects for the configuration
	// policy to check, create, modify, or delete on the cluster. 'object-templates' is an array
	// of objects, while 'object-templates-raw' is a string containing an array of objects in
	// YAML format. Only one of the two object-templates variables can be set in a given
	// configurationPolicy.
	ObjectTemplatesRaw string             `json:"object-templates-raw,omitempty"`
	EvaluationInterval EvaluationInterval `json:"evaluationInterval,omitempty"`
	// +kubebuilder:default:=None
	PruneObjectBehavior PruneObjectBehavior `json:"pruneObjectBehavior,omitempty"`
}

// ObjectTemplate describes how an object should look
type ObjectTemplate struct {
	// ComplianceType specifies whether it is: musthave, mustnothave, mustonlyhave
	ComplianceType ComplianceType `json:"complianceType"`

	MetadataComplianceType MetadataComplianceType `json:"metadataComplianceType,omitempty"`

	// ObjectDefinition defines required fields for the object
	// +kubebuilder:pruning:PreserveUnknownFields
	ObjectDefinition runtime.RawExtension `json:"objectDefinition,omitempty"`
}

// ConfigurationPolicyStatus defines the observed state of ConfigurationPolicy
type ConfigurationPolicyStatus struct {
	ComplianceState   ComplianceState  `json:"compliant,omitempty"`         // Compliant/NonCompliant/UnknownCompliancy
	CompliancyDetails []TemplateStatus `json:"compliancyDetails,omitempty"` // reason for non-compliancy
	// An ISO-8601 timestamp of the last time the policy was evaluated
	LastEvaluated string `json:"lastEvaluated,omitempty"`
	// The generation of the ConfigurationPolicy object when it was last evaluated
	LastEvaluatedGeneration int64 `json:"lastEvaluatedGeneration,omitempty"`
	// List of resources processed by the policy
	RelatedObjects []RelatedObject `json:"relatedObjects,omitempty"`
}

// CompliancePerClusterStatus contains aggregate status of other policies in cluster
type CompliancePerClusterStatus struct {
	AggregatePolicyStatus map[string]*ConfigurationPolicyStatus `json:"aggregatePoliciesStatus,omitempty"`
	ComplianceState       ComplianceState                       `json:"compliant,omitempty"`
	ClusterName           string                                `json:"clustername,omitempty"`
}

// ComplianceMap map to hold CompliancePerClusterStatus objects
type ComplianceMap map[string]*CompliancePerClusterStatus

// ResourceState genric description of a state
type ResourceState string

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Compliance state",type="string",JSONPath=".status.compliant"

// ConfigurationPolicy is the Schema for the configurationpolicies API
type ConfigurationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   *ConfigurationPolicySpec  `json:"spec,omitempty"`
	Status ConfigurationPolicyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ConfigurationPolicyList contains a list of ConfigurationPolicy
type ConfigurationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConfigurationPolicy `json:"items"`
}

// TemplateStatus hold the status result
type TemplateStatus struct {
	ComplianceState ComplianceState `json:"Compliant,omitempty"` // Compliant, NonCompliant, UnknownCompliancy
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []Condition `json:"conditions,omitempty"`

	Validity Validity `json:"Validity,omitempty"` // a template can be invalid if it has conflicting roles
}

// Validity describes if it is valid or not
type Validity struct {
	Valid  *bool  `json:"valid,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// ComplianceType describes whether we must or must not have a given resource
// +kubebuilder:validation:Enum=MustHave;Musthave;musthave;MustOnlyHave;Mustonlyhave;mustonlyhave;MustNotHave;Mustnothave;mustnothave
type ComplianceType string

const (
	// MustNotHave is an enforcement state to exclude a resource
	MustNotHave ComplianceType = "Mustnothave"

	// MustHave is an enforcement state to include a resource
	MustHave ComplianceType = "Musthave"

	// MustOnlyHave is an enforcement state to exclusively include a resource
	MustOnlyHave ComplianceType = "Mustonlyhave"
)

// MetadataComplianceType describes how to check compliance for the labels/annotations of a given object
// +kubebuilder:validation:Enum=MustHave;Musthave;musthave;MustOnlyHave;Mustonlyhave;mustonlyhave
type MetadataComplianceType string

// RelatedObject is the list of objects matched by this Policy resource.
type RelatedObject struct {
	//
	Object ObjectResource `json:"object,omitempty"`
	//
	Compliant string `json:"compliant,omitempty"`
	//
	Reason     string            `json:"reason,omitempty"`
	Properties *ObjectProperties `json:"properties,omitempty"`
}

// ObjectResource is an object identified by the policy as a resource that needs to be validated.
type ObjectResource struct {
	// Kind of the referent. More info:
	// https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
	Kind string `json:"kind,omitempty"`
	// API version of the referent.
	APIVersion string `json:"apiVersion,omitempty"`
	// Metadata values from the referent.
	Metadata ObjectMetadata `json:"metadata,omitempty"`
}

// ObjectMetadata contains the resource metadata for an object being processed by the policy
type ObjectMetadata struct {
	// Name of the referent. More info:
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
	Name string `json:"name,omitempty"`
	// Namespace of the referent. More info:
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/
	Namespace string `json:"namespace,omitempty"`
}

type ObjectProperties struct {
	// Whether the object was created by the parent policy
	CreatedByPolicy *bool `json:"createdByPolicy,omitempty"`
	// Store object UID to help track object ownership for deletion
	UID string `json:"uid,omitempty"`
}

func init() {
	SchemeBuilder.Register(&ConfigurationPolicy{}, &ConfigurationPolicyList{})
}
