/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2021 Red Hat, Inc.
 */

package manifests

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Prefix = "numacell-dp"
)

/*
 * The preferred approach would be to have the YAML manifests and to go embed them.
 * We are _intentionally_ not doing this, drifting from the best practice.
 * But the best practice applies for public-facing, library-wannabe packages.
 * This is a significantly different case:
 * - you should never ever consume these manifests outside e2e tests. Never ever.
 * - these manifests are doing nasty things, so they want to be as hidden as possible
 * - these manifests are expected to change rarely, if at all
 * - but more than else we want to hide the manifests and control their access as much
 *   as possible.
 *
 * We will reconsidering to bite the bullet and move them to plain YAML, go embed-able
 * files in the future.
 */

func ServiceAccount(namespace, name string) *corev1.ServiceAccount {
	sa := corev1.ServiceAccount{
		// TODO: avoid to hardcode values
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-sa",
			Namespace: namespace,
		},
	}
	return &sa
}

func Role(namespace, name string) *rbacv1.Role {
	ro := rbacv1.Role{
		// TODO: avoid to hardcode values
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-ro",
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"security.openshift.io",
				},
				ResourceNames: []string{
					"privileged",
				},
				Resources: []string{
					"securitycontextconstraints",
				},
				Verbs: []string{
					"use",
				},
			},
		},
	}
	return &ro
}

func RoleBinding(namespace, name string) *rbacv1.RoleBinding {
	rb := rbacv1.RoleBinding{
		// TODO: avoid to hardcode values
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-rb",
			Namespace: namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name + "-ro",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      name + "-sa",
				Namespace: namespace,
			},
		},
	}
	return &rb
}

func DaemonSet(nodeSelector map[string]string, namespace, name, saName, image string) *appsv1.DaemonSet {
	volName := "kubeletsockets"
	kubeletPath := "/var/lib/kubelet/device-plugins"
	podLabels := map[string]string{
		"name": name + "-pod",
	}
	hostPathDirectory := corev1.HostPathDirectory
	var zero int64
	var true_ bool = true
	ds := appsv1.DaemonSet{
		// TODO: avoid to hardcode values
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-ds",
			Namespace: namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: podLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: &zero,
					Containers: []corev1.Container{
						{
							Name:    name + "-cnt",
							Image:   image,
							Command: []string{"/bin/numacell"},
							Args: []string{
								"-alsologtostderr",
								"-v", "3",
							},
							ImagePullPolicy: corev1.PullAlways,
							SecurityContext: &corev1.SecurityContext{
								Privileged: &true_,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      volName,
									MountPath: kubeletPath,
								},
							},
						},
					},
					RestartPolicy:      corev1.RestartPolicyAlways,
					ServiceAccountName: saName,
					Volumes: []corev1.Volume{
						{
							Name: volName,
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: kubeletPath,
									Type: &hostPathDirectory,
								},
							},
						},
					},
				},
			},
		},
	}

	if nodeSelector != nil {
		ds.Spec.Template.Spec.NodeSelector = nodeSelector
	}
	return &ds
}
