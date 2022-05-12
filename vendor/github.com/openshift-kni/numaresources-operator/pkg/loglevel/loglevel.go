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
 * Copyright 2022 Red Hat, Inc.
 */

package loglevel

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/openshift-kni/numaresources-operator/pkg/flagcodec"
	operatorv1 "github.com/openshift/api/operator/v1"
)

// ToKlog converts LogLevel value into klog verboseness level according to operator/v1.LogLevel documentation
func ToKlog(level operatorv1.LogLevel) klog.Level {
	switch level {
	case operatorv1.Normal:
		return klog.Level(2)

	case operatorv1.Debug:
		return klog.Level(4)

	case operatorv1.Trace:
		return klog.Level(6)

	case operatorv1.TraceAll:
		return klog.Level(8)

	default:
		return klog.Level(2)
	}
}

func UpdatePodSpec(podSpec *corev1.PodSpec, level operatorv1.LogLevel) error {
	// TODO: better match by name than assume container#0 is the right one
	cnt := &podSpec.Containers[0]
	kLog := ToKlog(level)
	flags := flagcodec.ParseArgvKeyValue(cnt.Args)
	if flags == nil {
		return fmt.Errorf("cannot modify the arguments for container %s", cnt.Name)
	}
	flags.SetOption("--v", kLog.String())
	klog.InfoS("container klog level", "container", cnt.Name, "--v", kLog.String())
	cnt.Args = flags.Argv()
	return nil
}
