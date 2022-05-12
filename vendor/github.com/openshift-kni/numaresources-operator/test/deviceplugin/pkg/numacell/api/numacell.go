/*
Copyright 2020.

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

package api

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

const (
	NUMACellDevicePath        = "/dev/null"
	NUMACellResourceName      = "numacell"
	NUMACellResourceNamespace = "kni.node"

	NUMACellEnvironVarName = "KNI_NODE_CELL_ID"

	NUMACellDefaultDeviceCount = 15
)

func MakeResourceName(numacellid int) corev1.ResourceName {
	return corev1.ResourceName(fmt.Sprintf("%s/%s", NUMACellResourceNamespace, MakeDeviceID(numacellid))) // TODO
}

func MakeDeviceID(numacellid int) string {
	return fmt.Sprintf("%s%02d", NUMACellResourceName, numacellid)
}
