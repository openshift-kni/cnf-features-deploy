// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package v1

import (
	"errors"
	"unicode/utf8"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	// log is for logging in this package.
	policylog = logf.Log.WithName("policy-validating-webhook")
	errName   = errors.New("the combined length of the policy namespace and name " +
		"cannot exceed 62 characters")
)

func (r *Policy) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-policy-open-cluster-management-io-v1-policy,mutating=false,failurePolicy=Ignore,sideEffects=None,groups=policy.open-cluster-management.io,resources=policies,verbs=create,versions=v1,name=policy.open-cluster-management.io.webhook,admissionReviewVersions=v1

var _ webhook.Validator = &Policy{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Policy) ValidateCreate() (admission.Warnings, error) {
	policylog.Info("Validate policy creation request", "name", r.Name)

	return r.validateName()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Policy) ValidateUpdate(_ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Policy) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}

// validate the policy name and namespace length
func (r *Policy) validateName() (admission.Warnings, error) {
	policylog.Info("Validating the policy name through a validating webhook")

	// replicated policies don't need pass this validation
	if _, ok := r.GetLabels()["policy.open-cluster-management.io/root-policy"]; ok {
		return nil, nil
	}

	// 1 character for "."
	if (utf8.RuneCountInString(r.Name) + utf8.RuneCountInString(r.Namespace)) > 62 {
		return nil, errName
	}

	return nil, nil
}
