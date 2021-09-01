/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

type HelmRepo struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:Required
	URL string `json:"url"`
	// +kubebuilder:validation:Optional
	Username string `json:"username"`
	// +kubebuilder:validation:Optional
	Password string `json:"password"`
	// +kubebuilder:validation:Optional
	CertFile string `json:"certFile"`
	// +kubebuilder:validation:Optional
	KeyFile string `json:"keyFile"`
	// +kubebuilder:validation:Optional
	CAFile string `json:"caFile"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	InsecureSkipTLSverify bool `json:"insecure_skip_tls_verify"`
}

type HelmChart struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	// +kubebuilder:validation:Required
	Repository HelmRepo `json:"repository"`
	// +kubebuilder:validation:Optional
	Tags []string `json:"tags"`
}

func (in *HelmChart) DeepCopyInto(out *HelmChart) {
	*out = *in
	out.Repository = in.Repository
	if in.Tags != nil {
		in, out := &in.Tags, &out.Tags
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is a manually created deepcopy function, copying the receiver, creating a new HelmChart.
func (in *HelmChart) DeepCopy() *HelmChart {
	if in == nil {
		return nil
	}
	out := new(HelmChart)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is a manually created deepcopy function, copying the receiver, writing into out. in must be nonnil.
func (in *HelmRepo) DeepCopyInto(out *HelmRepo) {
	*out = *in
}

// DeepCopy is a manually created deepcopy function, copying the receiver, creating a new HelmRepo.
func (in *HelmRepo) DeepCopy() *HelmRepo {
	if in == nil {
		return nil
	}
	out := new(HelmRepo)
	in.DeepCopyInto(out)
	return out
}
