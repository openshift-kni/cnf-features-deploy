package main

import (
	"flag"
	"fmt"
	"os"

	infov1 "github.com/google/cadvisor/info/v1"
	"k8s.io/klog/v2"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/node-utils/pkg/machine"
)

type Args struct {
	RootDirectory string
	RawOutput     bool
	FormatK8S     bool
}

func main() {
	klog.InitFlags(nil)

	args := Args{
		FormatK8S: true,
	}

	flag.StringVar(&args.RootDirectory, "root-dir", args.RootDirectory, "use <arg> as root prefix - use this if run inside a container")
	flag.BoolVar(&args.RawOutput, "raw-output", args.RawOutput, "emit full output - including machine-identifiable parts")
	flag.BoolVar(&args.FormatK8S, "k8s", args.FormatK8S, "emit output in kubernetes machineinfo format")
	flag.Parse()

	if !args.FormatK8S {
		klog.InfoS("only the K8S machineinfo format is supported")
		os.Exit(1)
	}

	var minfo *infov1.MachineInfo
	var err error
	if args.RawOutput {
		minfo, err = machine.GetRaw(args.RootDirectory)
	} else {
		minfo, err = machine.Get(args.RootDirectory)
	}
	if err != nil {
		klog.ErrorS(err, "getting machine info", "root", args.RootDirectory)
		os.Exit(2)
	}

	data, err := machine.ToJSON(minfo)
	if err != nil {
		klog.ErrorS(err, "serializing machine info to JSON")
		os.Exit(4)
	}
	fmt.Print(data)
}
