/*
 * Copyright 2022 Red Hat, Inc.
 *
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
 */

package wait

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
)

func deletionStatusFromError(kind string, key ObjectKey, err error) (bool, error) {
	if err == nil {
		klog.Infof("%s %s still present", kind, key.String())
		return false, nil
	}
	if apierrors.IsNotFound(err) {
		klog.Infof("%s %s is gone", kind, key.String())
		return true, nil
	}
	klog.Warningf("failed to get the %s %s: %v", kind, key.String(), err)
	return false, err
}
