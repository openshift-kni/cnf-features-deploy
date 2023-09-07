package main

import (
	"flag"
	"syscall"
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

	mainThreadCPUs := selfCPUs.List()[0]
	klog.Infof("oslat main thread cpu: %d", mainThreadCPUs)

	siblings, err := node.GetCPUSiblings(mainThreadCPUs)
	if err != nil {
		klog.Fatalf("failed to get main thread CPU siblings: %v", err)
	}
	klog.Infof("oslat main thread's cpu siblings: %v", siblings)

	// siblings > 1 means Hyper-threading enabled
	if len(siblings) > 1 && selfCPUs.Size() == 2 {
		// one CPU should be used to run oslat's main thread.
		// the second is the sibling of the first one, which should be excluded from the list of the tested CPUs,
		// because it might cause to false spikes (noisy-neighbor issue).
		// the third one is the actual CPU to be tested, but due to SMT alignment restriction we need its sibling too.
		// four in total.
		klog.Fatalf("when hyper-threading enabled oslat pod requires at least 4 CPUs")
	}

	cpusForLatencyTest := selfCPUs.Difference(cpuset.New(siblings...))
	mainThreadCPUSet := cpuset.New(mainThreadCPUs)

	err = node.PrintInformation()
	if err != nil {
		klog.Fatalf("failed to print node information: %v", err)
	}

	if *oslatStartDelay > 0 {
		time.Sleep(time.Duration(*oslatStartDelay) * time.Second)
	}

	oslatArgs := []string{
		"oslat",
		"--duration", *runtime,
		"--rtprio", *rtPriority,
		"--cpu-list", cpusForLatencyTest.String(),
		"--cpu-main-thread", mainThreadCPUSet.String(),
	}

	klog.Infof("running oslat command with arguments %v", oslatArgs[1:])
	err = syscall.Exec(oslatBinary, oslatArgs, []string{})
	if err != nil {
		klog.Fatalf("failed to run oslat command %v", err)
	}
}
