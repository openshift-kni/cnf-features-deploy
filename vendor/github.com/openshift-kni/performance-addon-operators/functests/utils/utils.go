package utils

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"

	"k8s.io/klog"
)

const defaultExecTimeout = 2 * time.Minute

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
	// Create a new context and add a timeout to it
	ctx, cancel := context.WithTimeout(context.Background(), defaultExecTimeout)
	defer cancel() // The cancel should be deferred so resources are cleaned up

	out, err := exec.CommandContext(ctx, name, arg...).Output()
	klog.Infof("run command '%s %v' (err=%v):\n  stdout=%s\n", name, arg, err, out)

	// We want to check the context error to see if the timeout was executed.
	// The error returned by cmd.Output() will be OS specific based on what
	// happens when a process is killed.
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("command '%s %v' failed because of the timeout", name, arg)
	}

	if exitError, ok := err.(*exec.ExitError); ok {
		klog.Infof("run command '%s %v' (err=%v):\n  stderr=%s", name, arg, err, exitError.Stderr)
	}
	return out, err
}
