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
	"context"
	"errors"
	"net/url"

	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var ptpoperatorconfiglog = logf.Log.WithName("ptpoperatorconfig-resource")

var k8sclient client.Client

func (r *PtpOperatorConfig) SetupWebhookWithManager(mgr ctrl.Manager, client client.Client) error {
	k8sclient = client
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-ptp-openshift-io-v1-ptpoperatorconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=ptp.openshift.io,resources=ptpoperatorconfigs,verbs=update,versions=v1,name=vptpoperatorconfig.kb.io,admissionReviewVersions=v1

const (
	AmqScheme = "amqp"
	// storageTypeEmptyDir is used for developer tests to map pubsubstore volume to emptyDir
	storageTypeEmptyDir = "emptyDir"
)

func (r *PtpOperatorConfig) validate() error {
	eventConfig := r.Spec.EventConfig
	if eventConfig != nil && eventConfig.EnableEventPublisher {
		transportUrl, err := url.Parse(eventConfig.TransportHost)
		if err == nil && transportUrl.Scheme == AmqScheme {
			return nil
		}
		if eventConfig.StorageType == "" {
			// default to emptyDir to pass the check since cloud-event-proxy overwrites this to configMap for HTTP transport
			eventConfig.StorageType = storageTypeEmptyDir
		}
		if eventConfig.StorageType != storageTypeEmptyDir && !r.checkStorageClass(eventConfig.StorageType) {
			return errors.New("ptpEventConfig.storageType is set to StorageClass " + eventConfig.StorageType + " which does not exist")
		}
	}
	return nil
}

var _ webhook.Validator = &PtpOperatorConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *PtpOperatorConfig) ValidateCreate() (admission.Warnings, error) {
	ptpoperatorconfiglog.Info("validate create", "name", r.Name)
	return admission.Warnings{}, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *PtpOperatorConfig) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	ptpoperatorconfiglog.Info("validate update", "name", r.Name)
	if err := r.validate(); err != nil {
		return admission.Warnings{}, err
	}

	return admission.Warnings{}, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *PtpOperatorConfig) ValidateDelete() (admission.Warnings, error) {
	ptpoperatorconfiglog.Info("validate delete", "name", r.Name)
	return admission.Warnings{}, nil
}

func (r *PtpOperatorConfig) checkStorageClass(scName string) bool {

	scList := &storagev1.StorageClassList{}
	opts := []client.ListOption{}
	err := k8sclient.List(context.TODO(), scList, opts...)
	if err != nil {
		return false
	}

	for _, sc := range scList.Items {
		if sc.Name == scName {
			return true
		}
	}
	return false
}
