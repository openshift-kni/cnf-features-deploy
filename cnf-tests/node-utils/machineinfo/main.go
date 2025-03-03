package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jaypipes/ghw/pkg/topology"
	"k8s.io/klog"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/node-utils/pkg/environ"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/node-utils/pkg/machine"
)

func main() {
	klog.InitFlags(nil)

	env := environ.New()
	flag.StringVar(&env.Root.Sys, "sysfs", env.Root.Sys, "override sysfs path - use it if running inside a container")
	flag.Parse()

	machine, err := machine.Discover(env)
	if err != nil {
		env.Log.Error(err, "machine discover failed", "env", env)
		os.Exit(1)
	}

	// fixup ghw quirks
	machine.Topology.Architecture = topology.ARCHITECTURE_NUMA

	data, err := machine.ToJSON()
	if err != nil {
		env.Log.Error(err, "machine info JSON serialization failed")
		os.Exit(2)
	}
	fmt.Printf("%s", data)
}
