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
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObjectKey struct {
	Namespace string
	Name      string
}

func ObjectKeyFromObject(obj metav1.Object) ObjectKey {
	return ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}
}

func (ok ObjectKey) AsKey() types.NamespacedName {
	return types.NamespacedName{
		Namespace: ok.Namespace,
		Name:      ok.Name,
	}
}

func (ok ObjectKey) String() string {
	return fmt.Sprintf("%s/%s", ok.Namespace, ok.Name)
}

func WhileInPodPhase(cli client.Client, podNamespace, podName string, phase corev1.PodPhase, interval time.Duration, steps int) error {
	updatedPod := &corev1.Pod{}
	key := ObjectKey{Name: podName, Namespace: podNamespace}
	for step := 0; step < steps; step++ {
		time.Sleep(interval)

		klog.Infof("ensuring the pod %s keep being in phase %s %d/%d", key.String(), phase, step+1, steps)

		err := cli.Get(context.TODO(), client.ObjectKey{Namespace: podNamespace, Name: podName}, updatedPod)
		if err != nil {
			return err
		}

		if updatedPod.Status.Phase != phase {
			klog.Warningf("pod %s unexpected phase %q expected %q", key.String(), updatedPod.Status.Phase, string(phase))
			return fmt.Errorf("pod %s unexpected phase %q expected %q", key.String(), updatedPod.Status.Phase, string(phase))
		}
	}
	return nil
}

func ForPodPhase(cli client.Client, podNamespace, podName string, phase corev1.PodPhase, timeout time.Duration) (*corev1.Pod, error) {
	updatedPod := &corev1.Pod{}
	err := wait.PollImmediate(10*time.Second, timeout, func() (bool, error) {
		objKey := ObjectKey{Name: podName, Namespace: podNamespace}
		if err := cli.Get(context.TODO(), objKey.AsKey(), updatedPod); err != nil {
			klog.Warningf("failed to get the pod %#v: %v", objKey, err)
			return false, nil
		}

		if updatedPod.Status.Phase == phase {
			klog.Infof("pod %s reached phase %s", objKey.String(), string(phase))
			return true, nil
		}

		klog.Infof("pod %s phase %s desired %s", objKey.String(), string(updatedPod.Status.Phase), string(phase))
		return false, nil
	})
	return updatedPod, err
}

func ForPodDeleted(cli client.Client, podNamespace, podName string, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		pod := &corev1.Pod{}
		key := ObjectKey{Name: podName, Namespace: podNamespace}
		err := cli.Get(context.TODO(), key.AsKey(), pod)
		return deletionStatusFromError("Pod", key, err)
	})
}

func ForPodListAllRunning(cli client.Client, pods []*corev1.Pod) []*corev1.Pod {
	var failedLock sync.Mutex
	var failed []*corev1.Pod

	var wg sync.WaitGroup
	for _, pod := range pods {
		wg.Add(1)
		go func(pod *corev1.Pod) {
			defer wg.Done()

			klog.Infof("waiting for pod %q to be ready", pod.Name)

			_, err := ForPodPhase(cli, pod.Namespace, pod.Name, corev1.PodRunning, 3*time.Minute)
			if err != nil {
				// TODO: channels would be nicer
				failedLock.Lock()
				failed = append(failed, pod)
				failedLock.Unlock()
			}
		}(pod)
	}
	wg.Wait()
	return failed
}
