package main

import (
	"flag"
	"fmt"
	"k8s.io/klog"
	"os/exec"
	"strconv"
	"time"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/pod-utils/pkg/node"
)

const hwlatdetectBinary = "/usr/bin/hwlatdetect"
const cmdExecuter = "/usr/bin/python3"

func main() {
	klog.InitFlags(nil)

	threshold := flag.Int("threshold", 20, " value above which is considered an hardware latency")
	hardlimit := flag.Int("hardlimit", 20, " value above which the test is considered to fail")
	duration := flag.String("duration", "15s", "total time to test for hardware latency: <n>{smdw}")
	window := flag.Duration("window", time.Microsecond*10000000, "time between samples: <n>{usmss}")
	width := flag.Duration("width", time.Microsecond*950000, "time to actually measure: <n>{usmss}")
	hwlatdetectStartDelay := flag.Duration("hwlatdetect-start-delay", 0, "delay in second before running the hwlatdetect binary")

	flag.Parse()

	err := node.PrintInformation()
	if err != nil {
		klog.Fatalf("failed to print node information: %v", err)
	}

	time.Sleep(*hwlatdetectStartDelay)

	hwlatdetectArgs := []string{
		hwlatdetectBinary,
		"--threshold", strconv.Itoa(*threshold),
		"--hardlimit", strconv.Itoa(*hardlimit),
		"--duration", *duration,
		// convert values into a string with a single measurement unit
		// because the hwlatdetect tool doesn't know how to deal with other format
		// for example 5s is valid, but 5m4s and 5 isn't
		"--window", fmt.Sprintf("%dus", window.Microseconds()),
		"--width", fmt.Sprintf("%dus", width.Microseconds()),
	}

	klog.Infof("running the hwlatdetect command with arguments %v", hwlatdetectArgs)
	out, err := exec.Command(cmdExecuter, hwlatdetectArgs...).CombinedOutput()
	if err != nil {
		klog.Fatalf("failed to run hwlatdetect command; out: %s; err: %v", out, err)
	}

	klog.Infof("succeeded to run the hwlatdetect command: %s", out)
	klog.Flush()
}
