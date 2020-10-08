package utils

import (
	"os/exec"

	. "github.com/onsi/ginkgo"

	"k8s.io/klog"
)

func BeforeAll(fn func()) {
	first := true
	BeforeEach(func() {
		if first {
			fn()
			first = false
		}
	})
}

func ExecAndLogCommand(name string, arg ...string) ([]byte, error) {
	out, err := exec.Command(name, arg...).Output()
	klog.Infof("run command '%s %v' (err=%v):\n  stdout=%s\n", name, arg, err, out)
	if exitError, ok := err.(*exec.ExitError); ok {
		klog.Infof("run command '%s %v' (err=%v):\n  stderr=%s", name, arg, err, exitError.Stderr)
	}
	return out, err
}
