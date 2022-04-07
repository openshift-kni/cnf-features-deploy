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

package wait

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func deletionStatusFromError(kind string, key client.ObjectKey, err error) (bool, error) {
	if err == nil {
		klog.Infof("%s %#v still present", kind, key)
		return false, nil
	}
	if apierrors.IsNotFound(err) {
		klog.Infof("%s %#v is gone", kind, key)
		return true, nil
	}
	klog.Warningf("failed to get the %s %#v: %v", kind, key, err)
	return false, err
}
