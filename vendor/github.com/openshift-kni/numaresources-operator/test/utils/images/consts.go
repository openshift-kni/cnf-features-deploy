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

package images

const (
	RTETestImageCI   = "quay.io/openshift-kni/resource-topology-exporter:test-ci"
	SchedTestImageCI = "quay.io/openshift-kni/scheduler-plugins:test-ci"

	// NEVER EVER USE THIS OUTSIDE CI or (early) DEVELOPMENT ENVIRONMENTS
	NUMACellDevicePluginTestImageCI = "quay.io/openshift-kni/numacell-device-plugin:test-ci"

	//the default image used for test pods
	PauseImage = "gcr.io/google_containers/pause-amd64:3.0"
)
