package main

import (
	"flag"
	"k8s.io/klog"
	"os/exec"
	"strconv"
	"time"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/pod-utils/pkg/node"
)

const cyclictestBinary = "/usr/bin/cyclictest"

func main() {
	klog.InitFlags(nil)

	rtPriority := flag.String("rt-priority", "1", "specify the SCHED_FIFO priority (1-99)")
	duration := flag.String("duration", "15s", "specify a length for the test run. Append 'm', 'h', or 'd' to specify minutes, hours or days.")
	threads := flag.Int("threads", 1, "number of threads: if -1, number of threads will be max_cpus. without -t default = 1")
	histogram := flag.String("histogram", "30", "dump a latency histogram to stdout after the run US is the max latency time to be be tracked in microseconds")
	interval := flag.Int("interval", 1000, "base interval of thread in us default=1000")
	cyclictestStartDelay := flag.Duration("cyclictest-start-delay", 0, "delay in second before running the cyclictest binary")

	flag.Parse()

	selfCPUs, err := node.GetSelfCPUs()
	if err != nil {
		klog.Fatalf("failed to get self allowed CPUs: %v", err)
	}

	if *threads == -1 {
		*threads = selfCPUs.Size()
	}

	err = node.PrintInformation()
	if err != nil {
		klog.Fatalf("failed to print node information: %v", err)
	}

	time.Sleep(*cyclictestStartDelay)

	cyclictestArgs := []string{
		"-D", *duration,
		"-p", *rtPriority,
		"-t", strconv.Itoa(*threads),
		"-a", selfCPUs.String(),
		"-h", *histogram,
		"-i", strconv.Itoa(*interval),
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
