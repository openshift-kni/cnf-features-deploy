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

package nrosched

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nrtv1alpha1 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha1"
	nropv1alpha1 "github.com/openshift-kni/numaresources-operator/api/numaresourcesoperator/v1alpha1"
	"github.com/openshift-kni/numaresources-operator/pkg/status"
)

const (
	NROSchedObjectName = "numaresourcesscheduler"

	// scheduler
	ReasonScheduled        = "Scheduled"
	ReasonFailedScheduling = "FailedScheduling"
	// kubelet
	ReasonTopologyAffinityError = "TopologyAffinityError"

	// scheduler
	ErrorCannotAlignPod       = "cannot align pod"
	ErrorCannotAlignContainer = "cannot align container"
	// kubeket
	ErrorTopologyAffinityError = "Resources cannot be allocated with Topology locality"

	// component name
	kubeletName = "kubelet"
)

type eventChecker func(ev corev1.Event) bool

func checkPODEvents(k8sCli *kubernetes.Clientset, podNamespace, podName string, evCheck eventChecker) (bool, error) {
	By(fmt.Sprintf("checking events for pod %s/%s", podNamespace, podName))
	opts := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", podName),
		TypeMeta:      metav1.TypeMeta{Kind: "Pod"},
	}
	events, err := k8sCli.CoreV1().Events(podNamespace).List(context.TODO(), opts)
	if err != nil {
		klog.ErrorS(err, "cannot get events for pod %s/%s", podNamespace, podName)
		return false, err
	}
	if len(events.Items) == 0 {
		return false, fmt.Errorf("no event received for %s/%s", podNamespace, podName)
	}

	for _, item := range events.Items {
		klog.Infof("checking event: %s/%s [%s: %s - %s]", podNamespace, podName, item.ReportingController, item.Reason, item.Message)
		if evCheck(item) {
			klog.Infof("-> found relevant scheduling event for pod %s/%s: %v", podNamespace, podName, item)
			return true, nil
		}
	}
	klog.Warningf("Failed to find relevant scheduling event for pod %s/%s", podNamespace, podName)
	return false, nil
}

func CheckPODSchedulingFailed(k8sCli *kubernetes.Clientset, podNamespace, podName, schedulerName string) (bool, error) {
	isFailedScheduling := func(item corev1.Event) bool {
		return item.Reason == ReasonFailedScheduling && item.ReportingController == schedulerName
	}
	return checkPODEvents(k8sCli, podNamespace, podName, isFailedScheduling)
}

func CheckPODKubeletRejectWithTopologyAffinityError(k8sCli *kubernetes.Clientset, podNamespace, podName string) (bool, error) {
	isKubeletRejectForTopologyAffinityError := func(item corev1.Event) bool {
		if item.Reason != ReasonTopologyAffinityError {
			klog.Warningf("pod %s/%s reason %q expected %q", podNamespace, podName, item.Reason, ReasonTopologyAffinityError)
			return false
		}
		// kubernetes is quirky and the component naming is a bit of hard to grok
		if item.Source.Component != kubeletName {
			klog.Warningf("pod %s/%s controller %q expected %q", podNamespace, podName, item.Source.Component, kubeletName)
			return false
		}
		if !strings.Contains(item.Message, ErrorTopologyAffinityError) {
			klog.Warningf("pod %s/%s message %q expected %q", podNamespace, podName, item.Message, ErrorTopologyAffinityError)
			return false
		}
		return true
	}
	return checkPODEvents(k8sCli, podNamespace, podName, isKubeletRejectForTopologyAffinityError)
}

func CheckPODSchedulingFailedForAlignment(k8sCli *kubernetes.Clientset, podNamespace, podName, schedulerName, policy string) (bool, error) {
	var alignmentErr string
	if policy == string(nrtv1alpha1.SingleNUMANodeContainerLevel) {
		alignmentErr = ErrorCannotAlignContainer
	} else {
		alignmentErr = ErrorCannotAlignPod
	}

	isFailedSchedulingForAlignment := func(item corev1.Event) bool {
		if item.Reason != ReasonFailedScheduling {
			klog.Warningf("pod %s/%s reason %q expected %q", podNamespace, podName, item.Reason, ReasonFailedScheduling)
			return false
		}
		if item.ReportingController != schedulerName {
			klog.Warningf("pod %s/%s controller %q expected %q", podNamespace, podName, item.ReportingController, schedulerName)
			return false
		}
		// workaround kubelet race/confusing behaviour
		if !strings.Contains(item.Message, alignmentErr) {
			klog.Warningf("pod %s/%s message %q expected %q", podNamespace, podName, item.Message, alignmentErr)
			return false
		}
		return true
	}
	return checkPODEvents(k8sCli, podNamespace, podName, isFailedSchedulingForAlignment)
}

func CheckPODWasScheduledWith(k8sCli *kubernetes.Clientset, podNamespace, podName, schedulerName string) (bool, error) {
	isScheduledWith := func(item corev1.Event) bool {
		return item.Reason == ReasonScheduled && item.ReportingController == schedulerName
	}
	return checkPODEvents(k8sCli, podNamespace, podName, isScheduledWith)
}

func CheckNROSchedulerAvailable(cli client.Client, NUMAResourcesSchedObjName string) *nropv1alpha1.NUMAResourcesScheduler {
	nroSchedObj := &nropv1alpha1.NUMAResourcesScheduler{}
	Eventually(func() bool {
		By(fmt.Sprintf("checking %q for the condition Available=true", NUMAResourcesSchedObjName))

		err := cli.Get(context.TODO(), client.ObjectKey{Name: NROSchedObjectName}, nroSchedObj)
		if err != nil {
			klog.Warningf("failed to get the scheduler resource: %v", err)
			return false
		}

		cond := status.FindCondition(nroSchedObj.Status.Conditions, status.ConditionAvailable)
		if cond == nil {
			klog.Warningf("missing conditions in %v", nroSchedObj)
			return false
		}

		klog.Infof("condition: %v", cond)

		return cond.Status == metav1.ConditionTrue
	}, 5*time.Minute, 10*time.Second).Should(BeTrue(), "Scheduler condition did not become available")
	return nroSchedObj
}

func GetNROSchedulerName(cli client.Client, NUMAResourcesSchedObjName string) string {
	nroSchedObj := CheckNROSchedulerAvailable(cli, NUMAResourcesSchedObjName)
	return nroSchedObj.Status.SchedulerName
}
