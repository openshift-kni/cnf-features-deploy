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
	out, err := exec.Command(name, arg...).CombinedOutput()
	klog.Infof("run command '%s %v':\n  out=%s\n  err=%v", name, arg, out, err)
	return out, err
}
