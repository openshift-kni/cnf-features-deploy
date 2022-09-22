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

package noderesourcetopology

import (
	"fmt"
	"strings"

	nrtv1alpha1 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha1"
)

func ToString(resInfo nrtv1alpha1.ResourceInfo) string {
	return fmt.Sprintf("%s=%s/%s/%s", resInfo.Name, resInfo.Capacity.String(), resInfo.Allocatable.String(), resInfo.Available.String())
}

func ListToString(resInfoList []nrtv1alpha1.ResourceInfo) string {
	items := []string{}
	for _, resInfo := range resInfoList {
		items = append(items, ToString(resInfo))
	}
	return strings.Join(items, ",")
}
