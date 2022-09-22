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

package platform

import "strings"

type Platform string

const (
	Unknown    = Platform("Unknown")
	Kubernetes = Platform("Kubernetes")
	OpenShift  = Platform("OpenShift")
)

func (p Platform) String() string {
	return string(p)
}

func ParsePlatform(plat string) (Platform, bool) {
	plat = strings.ToLower(plat)
	switch plat {
	case "kubernetes":
		return Kubernetes, true
	case "openshift":
		return OpenShift, true
	default:
		return Unknown, false
	}
}
