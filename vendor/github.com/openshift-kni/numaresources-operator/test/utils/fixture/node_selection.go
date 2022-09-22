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

package fixture

import (
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
)

func PickNodeIndex(nodes []corev1.Node) (int, bool) {
	name, ok := os.LookupEnv("E2E_NROP_TARGET_NODE")
	if !ok {
		return 0, true // "random" default
	}
	for idx := range nodes {
		if nodes[idx].Name == name {
			klog.Infof("node %q found among candidates, picking", name)
			return idx, true
		}
	}
	klog.Infof("node %q not found among candidates, fall back to random one", name)
	return 0, false // "safe" default
}

func PopNodeName(nodeNames sets.String) (string, bool) {
	name, ok := os.LookupEnv("E2E_NROP_TARGET_NODE")
	if !ok {
		return nodeNames.PopAny()
	}

	if nodeNames.Has(name) {
		nodeNames.Delete(name)
		return name, true
	}

	klog.Infof("node %q not found among candidates, fall back to random one", name)
	return nodeNames.PopAny()
}
