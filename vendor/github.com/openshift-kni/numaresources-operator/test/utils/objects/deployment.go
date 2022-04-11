/*
Copyright 2022 The Kubernetes Authors.

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

package objects

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewTestDeployment(replicas int32, podLabels map[string]string, nodeSelector map[string]string, namespace, name, image string, command, args []string) *appsv1.Deployment {
	var zero int64
	podSpec := corev1.PodSpec{
		TerminationGracePeriodSeconds: &zero,
		Containers: []corev1.Container{
			{
				Name:    name + "-cnt",
				Image:   image,
				Command: command,
			},
		},
		RestartPolicy: corev1.RestartPolicyAlways,
	}
	dp := NewTestDeploymentWithPodSpec(replicas, podLabels, nodeSelector, namespace, name, podSpec)
	if nodeSelector != nil {
		dp.Spec.Template.Spec.NodeSelector = nodeSelector
	}
	return dp
}

func NewTestDeploymentWithPodSpec(replicas int32, podLabels map[string]string, nodeSelector map[string]string, namespace, name string, podSpec corev1.PodSpec) *appsv1.Deployment {
	dp := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: podLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: podSpec,
			},
		},
	}
	if nodeSelector != nil {
		dp.Spec.Template.Spec.NodeSelector = nodeSelector
	}
	return dp
}
