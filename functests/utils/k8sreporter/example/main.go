package main

import (
	"flag"
	"log"
	"os"
	"time"

	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/openshift-kni/cnf-features-deploy/functests/utils/k8sreporter"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "the kubeconfig path")
	report := flag.String("report", "report.log", "the file name used for the report")

	flag.Parse()

	addToScheme := func(s *runtime.Scheme) {
		mcfgv1.AddToScheme(s)
		promv1.AddToScheme(s)
	}

	filterPods := func(pod *v1.Pod) bool {
		// never filter
		return false
	}

	f, err := os.OpenFile(*report, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open the file: %v\n", err)
		return
	}
	defer f.Close()

	crs := []k8sreporter.CRData{
		k8sreporter.CRData{
			Cr: &mcfgv1.MachineConfigPoolList{},
		},
		k8sreporter.CRData{
			Cr: &promv1.ServiceMonitorList{},
		},
		k8sreporter.CRData{
			Cr:        &promv1.ServiceMonitorList{},
			Namespace: pointer.StringPtr("openshift-multus"),
		},
	}

	reporter := k8sreporter.New(*kubeconfig, addToScheme, filterPods, f, crs...)
	reporter.Dump(10 * time.Minute)
}
