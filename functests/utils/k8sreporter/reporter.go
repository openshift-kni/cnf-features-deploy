package k8sreporter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/types"
	corev1 "k8s.io/api/core/v1"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// FilterPods is a filter function to choose what pods to filter.
type FilterPods func(*v1.Pod) bool

// AddToScheme is a function for extend the reporter scheme and the CRs we are able to dump
type AddToScheme func(*runtime.Scheme)

// KubernetesReporter is a Ginkgo reporter that dumps info
// about configured kubernetes objects.
type KubernetesReporter struct {
	sync.Mutex
	clients    *clientSet
	dumpOutput io.Writer
	filterPods FilterPods
	crs        []CRData
}

// CRData represents a cr to dump
type CRData struct {
	Cr        runtime.Object
	Namespace *string
}

// New returns a new Kubernetes reporter from the given configuration.
func New(kubeconfig string, addToScheme AddToScheme, podsToLog FilterPods, dumpDestination io.Writer, crs ...CRData) (*KubernetesReporter, error) {
	crScheme := runtime.NewScheme()
	clientgoscheme.AddToScheme(crScheme)
	addToScheme(crScheme)

	clients, err := newClient(kubeconfig, crScheme)

	if err != nil {
		return nil, err
	}

	crsToDump := []CRData{}
	if crs != nil {
		crsToDump = crs[:]
	}

	return &KubernetesReporter{
		clients:    clients,
		dumpOutput: dumpDestination,
		filterPods: podsToLog,
		crs:        crsToDump,
	}, nil
}

// SpecSuiteWillBegin is the ginkgo callback on beginning of suite.
func (r *KubernetesReporter) SpecSuiteWillBegin(config config.GinkgoConfigType, summary *types.SuiteSummary) {

}

// BeforeSuiteDidRun is the ginkgo callback on running of a suite.
func (r *KubernetesReporter) BeforeSuiteDidRun(setupSummary *types.SetupSummary) {
	r.Cleanup()
}

// SpecWillRun is the ginkgo callback on running of a spec.
func (r *KubernetesReporter) SpecWillRun(specSummary *types.SpecSummary) {
}

// SpecDidComplete is the ginkgo callback on finishing a spec.
func (r *KubernetesReporter) SpecDidComplete(specSummary *types.SpecSummary) {
	r.Lock()
	defer r.Unlock()

	if !specSummary.HasFailureState() {
		return
	}
	fmt.Fprintln(r.dumpOutput, "Starting dump for failed spec", specSummary.ComponentTexts)
	r.Dump(specSummary.RunTime)
	fmt.Fprintln(r.dumpOutput, "Finished dump for failed spec")
}

// Dump dumps the relevant crs + pod logs
func (r *KubernetesReporter) Dump(duration time.Duration) {
	since := time.Now().Add(-duration).Add(-5 * time.Second)

	r.logNodes()
	r.logLogs(r.filterPods, since)
	r.logPods(r.filterPods)

	for _, cr := range r.crs {
		r.logCustomCR(cr.Cr, cr.Namespace)
	}
}

// Cleanup cleans up the current content of the artifactsDir
func (r *KubernetesReporter) Cleanup() {
}

func (r *KubernetesReporter) logPods(filterPods func(*corev1.Pod) bool) {
	fmt.Fprintf(r.dumpOutput, "Dumping pods definitions\n")

	pods, err := r.clients.Pods(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch pods: %v\n", err)
		return
	}

	for _, pod := range pods.Items {
		if filterPods(&pod) {
			continue
		}
		j, err := json.MarshalIndent(pod, "", "    ")
		if err != nil {
			fmt.Println("Failed to marshal pods", err)
			return
		}
		fmt.Fprintln(r.dumpOutput, string(j))
	}
}

func (r *KubernetesReporter) logNodes() {
	fmt.Fprintf(r.dumpOutput, "Dumping nodes\n")

	nodes, err := r.clients.Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch nodes: %v\n", err)
		return
	}

	j, err := json.MarshalIndent(nodes, "", "    ")
	if err != nil {
		fmt.Println("Failed to marshal nodes")
		return
	}
	fmt.Fprintln(r.dumpOutput, string(j))
}

func (r *KubernetesReporter) logLogs(filterPods FilterPods, since time.Time) {
	fmt.Fprintf(r.dumpOutput, "Dumping pods logs\n")

	pods, err := r.clients.Pods(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch pods: %v\n", err)
		return
	}

	for _, pod := range pods.Items {
		if filterPods(&pod) {
			continue
		}
		for _, container := range pod.Spec.Containers {
			logStart := metav1.NewTime(since)
			logs, err := r.clients.Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{Container: container.Name, SinceTime: &logStart}).DoRaw(context.Background())
			if err == nil {
				fmt.Fprintf(r.dumpOutput, "Dumping logs for pod %s-%s-%s\n", pod.Namespace, pod.Name, container.Name)
				fmt.Fprintln(r.dumpOutput, string(logs))
			}
		}
	}
}

func (r *KubernetesReporter) logCustomCR(cr runtime.Object, namespace *string) {

	if namespace != nil {
		fmt.Fprintf(r.dumpOutput, "Dumping %T in namespace %s\n", cr, *namespace)
	} else {
		fmt.Fprintf(r.dumpOutput, "Dumping %T\n", cr)
	}

	options := []runtimeclient.ListOption{}
	if namespace != nil {
		options = append(options, runtimeclient.InNamespace(*namespace))
	}
	err := r.clients.List(context.Background(),
		cr,
		options...)

	if err != nil {
		// this can be expected if we are reporting a feature we did not install the operator for
		fmt.Fprintf(r.dumpOutput, "Failed to fetch %T: %v\n", cr, err)
		return
	}

	j, err := json.MarshalIndent(cr, "", "    ")
	if err != nil {
		fmt.Fprintf(r.dumpOutput, "Failed to marshal %T\n", cr)
		return
	}
	fmt.Fprintln(r.dumpOutput, string(j))
}

// AfterSuiteDidRun is the ginkgo callback after suite run.
func (r *KubernetesReporter) AfterSuiteDidRun(setupSummary *types.SetupSummary) {

}

// SpecSuiteDidEnd is the ginkgo callback after end of suite.
func (r *KubernetesReporter) SpecSuiteDidEnd(summary *types.SuiteSummary) {

}
