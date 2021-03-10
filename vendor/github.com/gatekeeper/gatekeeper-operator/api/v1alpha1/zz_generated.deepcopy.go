// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuditConfig) DeepCopyInto(out *AuditConfig) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	if in.AuditInterval != nil {
		in, out := &in.AuditInterval, &out.AuditInterval
		*out = new(metav1.Duration)
		**out = **in
	}
	if in.ConstraintViolationLimit != nil {
		in, out := &in.ConstraintViolationLimit, &out.ConstraintViolationLimit
		*out = new(uint64)
		**out = **in
	}
	if in.AuditFromCache != nil {
		in, out := &in.AuditFromCache, &out.AuditFromCache
		*out = new(AuditFromCacheMode)
		**out = **in
	}
	if in.AuditChunkSize != nil {
		in, out := &in.AuditChunkSize, &out.AuditChunkSize
		*out = new(uint64)
		**out = **in
	}
	if in.LogLevel != nil {
		in, out := &in.LogLevel, &out.LogLevel
		*out = new(LogLevelMode)
		**out = **in
	}
	if in.EmitAuditEvents != nil {
		in, out := &in.EmitAuditEvents, &out.EmitAuditEvents
		*out = new(EmitEventsMode)
		**out = **in
	}
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = new(v1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuditConfig.
func (in *AuditConfig) DeepCopy() *AuditConfig {
	if in == nil {
		return nil
	}
	out := new(AuditConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Gatekeeper) DeepCopyInto(out *Gatekeeper) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Gatekeeper.
func (in *Gatekeeper) DeepCopy() *Gatekeeper {
	if in == nil {
		return nil
	}
	out := new(Gatekeeper)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Gatekeeper) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GatekeeperList) DeepCopyInto(out *GatekeeperList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Gatekeeper, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GatekeeperList.
func (in *GatekeeperList) DeepCopy() *GatekeeperList {
	if in == nil {
		return nil
	}
	out := new(GatekeeperList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *GatekeeperList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GatekeeperSpec) DeepCopyInto(out *GatekeeperSpec) {
	*out = *in
	if in.Image != nil {
		in, out := &in.Image, &out.Image
		*out = new(ImageConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.Audit != nil {
		in, out := &in.Audit, &out.Audit
		*out = new(AuditConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.ValidatingWebhook != nil {
		in, out := &in.ValidatingWebhook, &out.ValidatingWebhook
		*out = new(WebhookMode)
		**out = **in
	}
	if in.MutatingWebhook != nil {
		in, out := &in.MutatingWebhook, &out.MutatingWebhook
		*out = new(WebhookMode)
		**out = **in
	}
	if in.Webhook != nil {
		in, out := &in.Webhook, &out.Webhook
		*out = new(WebhookConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.PodAnnotations != nil {
		in, out := &in.PodAnnotations, &out.PodAnnotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GatekeeperSpec.
func (in *GatekeeperSpec) DeepCopy() *GatekeeperSpec {
	if in == nil {
		return nil
	}
	out := new(GatekeeperSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GatekeeperStatus) DeepCopyInto(out *GatekeeperStatus) {
	*out = *in
	if in.AuditConditions != nil {
		in, out := &in.AuditConditions, &out.AuditConditions
		*out = make([]StatusCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.WebhookConditions != nil {
		in, out := &in.WebhookConditions, &out.WebhookConditions
		*out = make([]StatusCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GatekeeperStatus.
func (in *GatekeeperStatus) DeepCopy() *GatekeeperStatus {
	if in == nil {
		return nil
	}
	out := new(GatekeeperStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageConfig) DeepCopyInto(out *ImageConfig) {
	*out = *in
	if in.Image != nil {
		in, out := &in.Image, &out.Image
		*out = new(string)
		**out = **in
	}
	if in.ImagePullPolicy != nil {
		in, out := &in.ImagePullPolicy, &out.ImagePullPolicy
		*out = new(v1.PullPolicy)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageConfig.
func (in *ImageConfig) DeepCopy() *ImageConfig {
	if in == nil {
		return nil
	}
	out := new(ImageConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StatusCondition) DeepCopyInto(out *StatusCondition) {
	*out = *in
	in.LastProbeTime.DeepCopyInto(&out.LastProbeTime)
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StatusCondition.
func (in *StatusCondition) DeepCopy() *StatusCondition {
	if in == nil {
		return nil
	}
	out := new(StatusCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WebhookConfig) DeepCopyInto(out *WebhookConfig) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	if in.LogLevel != nil {
		in, out := &in.LogLevel, &out.LogLevel
		*out = new(LogLevelMode)
		**out = **in
	}
	if in.EmitAdmissionEvents != nil {
		in, out := &in.EmitAdmissionEvents, &out.EmitAdmissionEvents
		*out = new(EmitEventsMode)
		**out = **in
	}
	if in.FailurePolicy != nil {
		in, out := &in.FailurePolicy, &out.FailurePolicy
		*out = new(admissionregistrationv1.FailurePolicyType)
		**out = **in
	}
	if in.NamespaceSelector != nil {
		in, out := &in.NamespaceSelector, &out.NamespaceSelector
		*out = new(metav1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = new(v1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WebhookConfig.
func (in *WebhookConfig) DeepCopy() *WebhookConfig {
	if in == nil {
		return nil
	}
	out := new(WebhookConfig)
	in.DeepCopyInto(out)
	return out
}
