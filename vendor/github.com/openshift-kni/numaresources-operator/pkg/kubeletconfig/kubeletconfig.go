/*
Copyright 2022.

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

package kubeletconfig

import (
	"encoding/json"

	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"

	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
)

func MCOKubeletConfToKubeletConf(mcoKc *mcov1.KubeletConfig) (*kubeletconfigv1beta1.KubeletConfiguration, error) {
	kc := &kubeletconfigv1beta1.KubeletConfiguration{}
	err := json.Unmarshal(mcoKc.Spec.KubeletConfig.Raw, kc)
	return kc, err
}

func KubeletConfToMCKubeletConf(kcObj *kubeletconfigv1beta1.KubeletConfiguration, kcAsMc *mcov1.KubeletConfig) error {
	rawKc, err := json.Marshal(kcObj)
	kcAsMc.Spec.KubeletConfig.Raw = rawKc
	return err
}
