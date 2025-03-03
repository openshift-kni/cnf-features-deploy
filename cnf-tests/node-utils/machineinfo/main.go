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

type Args struct {
	FormatGHW bool
}

func main() {
	klog.InitFlags(nil)

	args := Args{}
	env := environ.New()
	flag.StringVar(&env.Root.Sys, "sysfs", env.Root.Sys, "override sysfs path - use it if running inside a container")
	flag.BoolVar(&args.FormatGHW, "ghw", true, "emit machineinfo in the GHW format")
	flag.Parse()

	if !args.FormatGHW {
		env.Log.Info("only the GHW format is supported currently")
		os.Exit(1)
	}

	machine, err := machine.Discover(env)
	if err != nil {
		env.Log.Error(err, "machine discover failed", "env", env)
		os.Exit(2)
	}

	// fixup ghw quirks
	machine.Topology.Architecture = topology.ARCHITECTURE_NUMA

	data, err := machine.ToJSON()
	if err != nil {
		env.Log.Error(err, "machine info JSON serialization failed")
		os.Exit(4)
	}
	fmt.Printf("%s", data)
}
