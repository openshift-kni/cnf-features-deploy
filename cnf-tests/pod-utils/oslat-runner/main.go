package main

import (
	"flag"
	"os/exec"
	"time"

	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/pod-utils/pkg/node"
)

const oslatBinary = "/usr/bin/oslat"

func main() {
	klog.InitFlags(nil)

	var oslatStartDelay = flag.Int("oslat-start-delay", 0, "Delay in second before running the oslat binary, can be useful to be sure that the CPU manager excluded the pinned CPUs from the default CPU pool")
	var rtPriority = flag.String("rt-priority", "1", "Specify the SCHED_FIFO priority (1-99)")
	var runtime = flag.String("runtime", "10m", "Specify test duration, e.g., 60, 20m, 2H")

	flag.Parse()

	selfCPUs, err := node.GetSelfCPUs()
	if err != nil {
		klog.Fatalf("failed to get self allowed CPUs: %v", err)
	}

	if selfCPUs.Size() < 2 {
		klog.Fatalf("the amount of requested CPUs less than 2, the oslat requires at least 2 CPUs to run")
	}

	mainThreadCPUs := selfCPUs.ToSlice()[0]
	siblings, err := node.GetCPUSiblings(mainThreadCPUs)
	if err != nil {
		klog.Fatalf("failed to get main thread CPU siblings: %v", err)
	}
	cpusForLatencyTest := selfCPUs.Difference(cpuset.NewCPUSet(siblings...))
	mainThreadCPUSet := cpuset.NewCPUSet(mainThreadCPUs)

	err = node.PrintInformation()
	if err != nil {
		klog.Fatalf("failed to print node information: %v", err)
	}

	if *oslatStartDelay > 0 {
		time.Sleep(time.Duration(*oslatStartDelay) * time.Second)
	}

	oslatArgs := []string{
		"--duration", *runtime,
		"--rtprio", *rtPriority,
		"--cpu-list", cpusForLatencyTest.String(),
		"--cpu-main-thread", mainThreadCPUSet.String(),
	}

	klog.Infof("Running the oslat command with arguments %v", oslatArgs)
	out, err := exec.Command(oslatBinary, oslatArgs...).CombinedOutput()
	if err != nil {
		klog.Fatalf("failed to run oslat command; out: %s; err: %v", out, err)
	}

	klog.Infof("Succeeded to run the oslat command: %s", out)
	klog.Flush()
}
