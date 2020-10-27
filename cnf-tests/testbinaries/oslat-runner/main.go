package main

import (
	"flag"
	"os/exec"
	"strings"
	"time"

	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
)

func main() {
	klog.InitFlags(nil)

	var oslatStartDelay = flag.Int("oslat-start-delay", 0, "Delay in second before running the oslat binary, can be useful to be sure that the CPU manager excluded the pinned CPUs from the default CPU pool")
	var rtPriority = flag.String("rt-priority", "1", "Specify the SCHED_FIFO priority (1-99)")
	var runtime = flag.String("runtime", "10m", "Specify test duration, e.g., 60, 20m, 2H")

	flag.Parse()

	selfCPUs, err := getSelfCPUs()
	if err != nil {
		klog.Fatalf("failed to get self allowed CPUs: %v", err)
	}

	if selfCPUs.Size() < 2 {
		klog.Fatalf("the amount of requested CPUs less than 2, the oslat requires at least 2 CPUs to run")
	}

	mainThreadCPUSet := cpuset.NewCPUSet(selfCPUs.ToSlice()[0])
	updatedSelfCPUs := selfCPUs.Difference(mainThreadCPUSet)

	printNodeInformation()

	if *oslatStartDelay > 0 {
		time.Sleep(time.Duration(*oslatStartDelay) * time.Second)
	}

	oslatArgs := []string{
		"--runtime", *runtime,
		"--rtprio", *rtPriority,
		"--cpu-list", updatedSelfCPUs.String(),
		"--cpu-main-thread", mainThreadCPUSet.String(),
	}

	klog.Infof("Running the oslat command with arguments %v", oslatArgs)
	out, err := exec.Command("/usr/bin/oslat", oslatArgs...).CombinedOutput()
	if err != nil {
		klog.Fatalf("failed to run oslat command; out: %s; err: %v", out, err)
	}

	klog.Infof("Succeeded to run the oslat command: %s", out)
	klog.Flush()
}

func getSelfCPUs() (*cpuset.CPUSet, error) {
	cmd := exec.Command("/bin/sh", "-c", "grep Cpus_allowed_list /proc/self/status | cut -f2")
	out, err := cmd.CombinedOutput()
	if err != nil {
		klog.Errorf("failed to run command, out: %s; err: %v", out, err)
		return nil, err
	}

	cpus, err := cpuset.Parse(strings.Trim(string(out), "\n"))
	if err != nil {
		return nil, err
	}

	return &cpus, nil
}

func printNodeInformation() {
	out, err := exec.Command("cat", "/proc/cmdline").CombinedOutput()
	if err != nil {
		klog.Fatalf("failed to get cmdline, out: %s, err: %v", out, err)
	}
	klog.Infof("Environment information: /proc/cmdline: %s", string(out))

	out, err = exec.Command("uname", "-nr").CombinedOutput()
	if err != nil {
		klog.Fatalf("failed to get kernel version, out: %s, err: %v", out, err)
	}
	klog.Infof("Environment information: uname -nr: %s", string(out))
}
