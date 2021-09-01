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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	helmerv1beta1 "github.com/openshift-psap/special-resource-operator/pkg/helmer/api/v1beta1"
)

// SpecialResourceImages defines the observed state of SpecialResource
type SpecialResourceImages struct {
	Name       string                 `json:"name"`
	Kind       string                 `json:"kind"`
	Namespace  string                 `json:"namespace"`
	PullSecret string                 `json:"pullsecret,omitempty"`
	Paths      []SpecialResourcePaths `json:"path"`
}

// SpecialResourceClaims defines the observed state of SpecialResource
type SpecialResourceClaims struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
}

// SpecialResourcePaths defines the observed state of SpecialResource
type SpecialResourcePaths struct {
	SourcePath     string `json:"sourcePath"`
	DestinationDir string `json:"destinationDir"`
}

// SpecialResourceArtifacts defines the observed state of SpecialResource
type SpecialResourceArtifacts struct {
	// +kubebuilder:validation:Optional
	HostPaths []SpecialResourcePaths `json:"hostPaths,omitempty"`
	// +kubebuilder:validation:Optional
	Images []SpecialResourceImages `json:"images,omitempty"`
	// +kubebuilder:validation:Optional
	Claims []SpecialResourceClaims `json:"claims,omitempty"`
}

// SpecialResourceBuildArgs defines the observed state of SpecialResource
type SpecialResourceBuildArgs struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SpecialResourceConfiguration defines the observed state of SpecialResource
type SpecialResourceConfiguration struct {
	Name  string   `json:"name"`
	Value []string `json:"value"`
}

// SpecialResourceGit defines the observed state of SpecialResource
type SpecialResourceGit struct {
	Ref string `json:"ref"`
	Uri string `json:"uri"`
}

// SpecialResourceSource defines the observed state of SpecialResource
type SpecialResourceSource struct {
	Git SpecialResourceGit `json:"git,omitempty"`
}

// SpecialResourceDriverContainer defines the desired state of SpecialResource
type SpecialResourceDriverContainer struct {
	// +kubebuilder:validation:Optional
	Source SpecialResourceSource `json:"source,omitempty"`
	// +kubebuilder:validation:Optional
	Artifacts SpecialResourceArtifacts `json:"artifacts,omitempty"`
}

// SpecialResourceSpec defines the desired state of SpecialResource
type SpecialResourceSpec struct {
	// +kubebuilder:validation:Required
	Chart helmerv1beta1.HelmChart `json:"chart"`
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`
	// +kubebuilder:validation:Optional
	ForceUpgrade bool `json:"forceUpgrade"`
	// +kubebuilder:validation:Optional
	Debug bool `json:"debug"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:EmbeddedResource
	Set unstructured.Unstructured `json:"set,omitempty"`
	// +kubebuilder:validation:Optional
	DriverContainer SpecialResourceDriverContainer `json:"driverContainer,omitempty"`
	// +kubebuilder:validation:Optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// +kubebuilder:validation:Optional
	Dependencies []SpecialResourceDependency `json:"dependencies,omitempty"`
}

// SpecialResourceDependency a dependent helm chart
type SpecialResourceDependency struct {
	helmerv1beta1.HelmChart `json:"chart,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:EmbeddedResource
	Set unstructured.Unstructured `json:"set,omitempty"`
}

// SpecialResourceStatus defines the observed state of SpecialResource
type SpecialResourceStatus struct {
	State string `json:"state"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SpecialResource is the Schema for the specialresources API
// +kubebuilder:resource:path=specialresources,scope=Cluster
// +kubebuilder:resource:path=specialresources,scope=Cluster,shortName=sr
type SpecialResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +kubebuilder:validation:Required
	Spec   SpecialResourceSpec   `json:"spec,omitempty"`
	Status SpecialResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SpecialResourceList contains a list of SpecialResource
type SpecialResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SpecialResource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SpecialResource{}, &SpecialResourceList{})
}
