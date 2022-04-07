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

package v1alpha1

// This is borrowed from the kubernetes source, because controller-gen
// complains about the kube native type:
// encountered struct field "Namespace" without JSON tag in type "NamespacedName"
// at least until kube catches up, we just inline this simple struct here.

// NamespacedName comprises a resource name, with a mandatory namespace,
// rendered as "<namespace>/<name>".
type NamespacedName struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

const (
	Separator = '/'
)

// String returns the general purpose string representation
func (n NamespacedName) String() string {
	return n.Namespace + string(Separator) + n.Name
}
