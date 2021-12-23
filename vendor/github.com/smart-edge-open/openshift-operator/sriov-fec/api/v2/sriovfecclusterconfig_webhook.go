// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2021 Intel Corporation

package v2

import (
	"github.com/smart-edge-open/openshift-operator/common/pkg/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var sriovfecclusterconfiglog = utils.NewLogger()

func (in *SriovFecClusterConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(in).Complete()
}

//+kubebuilder:webhook:path=/validate-sriovfec-intel-com-v2-sriovfecclusterconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=sriovfec.intel.com,resources=sriovfecclusterconfigs,verbs=create;update,versions=v2,name=vsriovfecclusterconfig.kb.io,admissionReviewVersions={v1}

var _ webhook.Validator = &SriovFecClusterConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *SriovFecClusterConfig) ValidateCreate() error {
	sriovfecclusterconfiglog.WithField("name", in.Name).Info("validate create")
	if errs := validate(in.Spec); len(errs) != 0 {
		return apierrors.NewInvalid(schema.GroupKind{Group: "sriovfec.intel.com", Kind: "SriovFecClusterConfig"}, in.Name, errs)
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *SriovFecClusterConfig) ValidateUpdate(_ runtime.Object) error {
	sriovfecclusterconfiglog.WithField("name", in.Name).Info("validate update")
	if errs := validate(in.Spec); len(errs) != 0 {
		return apierrors.NewInvalid(schema.GroupKind{Group: "sriovfec.intel.com", Kind: "SriovFecClusterConfig"}, in.Name, errs)
	}
	return nil
}

func validate(spec SriovFecClusterConfigSpec) (errs field.ErrorList) {

	validators := []func(spec SriovFecClusterConfigSpec) field.ErrorList{
		nodesFieldValidator,
		ambiguousBBDevConfigValidator,
		n3000LinkQueuesValidator,
		acc100VfAmountValidator,
		acc100NumQueueGroupsValidator,
	}

	for _, validate := range validators {
		errs = append(errs, validate(spec)...)
	}

	return errs
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *SriovFecClusterConfig) ValidateDelete() error {
	sriovfecclusterconfiglog.WithField("name", in.Name).Info("validate delete")
	//do nothing
	return nil
}

func nodesFieldValidator(spec SriovFecClusterConfigSpec) field.ErrorList {
	if spec.Nodes != nil {
		return field.ErrorList{
			field.Invalid(
				field.NewPath("spec", "nodes"),
				spec.Nodes,
				"using deprecated field 'nodes', which is no longer supported. Use 'acceleratorSelector' and 'nodeSelector' instead.",
			),
		}
	}
	return nil
}

func ambiguousBBDevConfigValidator(spec SriovFecClusterConfigSpec) (errs field.ErrorList) {
	if spec.PhysicalFunction.BBDevConfig.N3000 != nil && spec.PhysicalFunction.BBDevConfig.ACC100 != nil {
		err := field.Forbidden(
			field.NewPath("spec").Child("physicalFunction").Child("bbDevConfig"),
			"specified bbDevConfig cannot contain acc100 and n3000 configuration in the same time")
		errs = append(errs, err)
	}
	return
}

func n3000LinkQueuesValidator(spec SriovFecClusterConfigSpec) (errs field.ErrorList) {

	validateN3000Queues := func(qID *field.Path, queues UplinkDownlinkQueues) *field.Error {
		total := queues.VF0 + queues.VF1 + queues.VF2 + queues.VF3 + queues.VF4 + queues.VF5 + queues.VF5 + queues.VF6 + queues.VF7
		if total > 32 {
			return field.Invalid(qID, total, "sum of all specified queues must be no more than 32")
		}
		return nil
	}

	if n3000Config := spec.PhysicalFunction.BBDevConfig.N3000; n3000Config != nil {
		queuePath := field.NewPath("spec").
			Child("physicalFunction").
			Child("bbDevConfig", "n3000", "uplink", "queues")

		if err := validateN3000Queues(queuePath, n3000Config.Uplink.Queues); err != nil {
			errs = append(errs, err)
		}

		queuePath = field.NewPath("spec").
			Child("physicalFunction").
			Child("bbDevConfig", "n3000", "downlink", "queues")

		if err := validateN3000Queues(queuePath, n3000Config.Downlink.Queues); err != nil {
			errs = append(errs, err)
		}
	}

	return
}

func acc100VfAmountValidator(spec SriovFecClusterConfigSpec) (errs field.ErrorList) {

	validate := func(accConfig *ACC100BBDevConfig, vfAmount int, path *field.Path) *field.Error {
		if accConfig == nil {
			return nil
		}

		if vfAmount == 0 && accConfig.NumVfBundles != 0 {
			return field.Invalid(
				path,
				accConfig.NumVfBundles,
				"non zero value of numVfBundles cannot be accepted when physicalFunction.vfAmount equals 0")
		}

		if vfAmount != accConfig.NumVfBundles {
			return field.Invalid(
				path,
				accConfig.NumVfBundles,
				"value should be the same as physicalFunction.vfAmount")
		}
		return nil
	}

	if err := validate(
		spec.PhysicalFunction.BBDevConfig.ACC100,
		spec.PhysicalFunction.VFAmount,
		field.NewPath("spec").Child("physicalFunction").Child("bbDevConfig", "acc100", "numVfBundles"),
	); err != nil {
		errs = append(errs, err)
	}

	return
}

func acc100NumQueueGroupsValidator(spec SriovFecClusterConfigSpec) (errs field.ErrorList) {

	validate := func(accConfig *ACC100BBDevConfig, path *field.Path) *field.Error {
		if accConfig == nil {
			return nil
		}

		downlink4g := accConfig.Downlink4G.NumQueueGroups
		uplink4g := accConfig.Uplink4G.NumQueueGroups
		downlink5g := accConfig.Downlink5G.NumQueueGroups
		uplink5g := accConfig.Uplink5G.NumQueueGroups

		if sum := downlink5g + uplink5g + downlink4g + uplink4g; sum > 8 {
			return field.Invalid(
				field.NewPath("spec", "physicalFunction", "bbDevConfig", "acc100", "[downlink4G|uplink4G|downlink5G|uplink5G]", "numQueueGroups"),
				sum,
				"sum of all numQueueGroups should not be greater than 8",
			)
		}
		return nil
	}

	if err := validate(spec.PhysicalFunction.BBDevConfig.ACC100, field.NewPath("spec", "physicalFunction", "bbDevConfig", "acc100", "[downlink4G|uplink4G|downlink5G|uplink5G]", "numQueueGroups")); err != nil {
		errs = append(errs, err)
	}

	return
}
