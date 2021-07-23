package node

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"

	"golang.org/x/sys/unix"
)

func GetSelfCPUs() (*cpuset.CPUSet, error) {
	cmd := exec.Command("/bin/sh", "-c", "grep Cpus_allowed_list /proc/self/status | cut -f2")
	out, err := cmd.CombinedOutput()
	if err != nil {
		klog.Errorf("failed to run command, out: %s; err: %v", out, err)
		return nil, err
	}

	cpus, err := cpuset.Parse(strings.Trim(string(out), "\n"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse cpuset; err:%v", err)
	}

	return &cpus, nil
}

func PrintInformation() error {
	out, err := ioutil.ReadFile("/proc/cmdline")
	if err != nil {
		klog.Errorf("failed to read file /proc/cmdline")
		return err
	}
	klog.Infof("Environment information: /proc/cmdline: %s", string(out))

	uname := &unix.Utsname{}
	if err = unix.Uname(uname); err != nil {
		klog.Errorf("failed get system information")
		return err
	}
	klog.Infof("Environment information: kernel version %s", uname.Release)
	return nil
}
