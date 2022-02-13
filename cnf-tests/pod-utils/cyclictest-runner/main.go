package main

import (
	"flag"
	"os/exec"
	"strconv"
	"time"

	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/pod-utils/pkg/node"
)

const cyclictestBinary = "/usr/bin/cyclictest"

func main() {
	klog.InitFlags(nil)

	rtPriority := flag.String("rt-priority", "95", "specify the SCHED_FIFO priority (1-99)")
	duration := flag.String("duration", "15s", "specify a length for the test run. Append 'm', 'h', or 'd' to specify minutes, hours or days.")
	histogram := flag.String("histogram", "30", "dump a latency histogram to stdout after the run US is the max latency time to be be tracked in microseconds")
	interval := flag.Int("interval", 1000, "base interval of thread in us default=1000")
	cyclictestStartDelay := flag.Int("cyclictest-start-delay", 0, "delay in second before running the cyclictest binary")

	flag.Parse()

	selfCPUs, err := node.GetSelfCPUs()
	if err != nil {
		klog.Fatalf("failed to get self allowed CPUs: %v", err)
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

	if *cyclictestStartDelay > 0 {
		time.Sleep(time.Duration(*cyclictestStartDelay) * time.Second)
	}

	cyclictestArgs := []string{
		"--duration", *duration,
		"--priority", *rtPriority,
		"--threads", strconv.Itoa(cpusForLatencyTest.Size()),
		"--affinity", cpusForLatencyTest.String(),
		"--histogram", *histogram,
		"--interval", strconv.Itoa(*interval),
		"--mlockall",
		"--mainaffinity", mainThreadCPUSet.String(),
		"--smi",
		"--quiet",
	}

	klog.Infof("running the cyclictest command with arguments %v", cyclictestArgs)
	out, err := exec.Command(cyclictestBinary, cyclictestArgs...).CombinedOutput()
	if err != nil {
		klog.Fatalf("failed to run cyclictest command; out: %s; err: %v", out, err)
	}

	klog.Infof("succeeded to run the cyclictest command: %s", out)
	klog.Flush()
}
