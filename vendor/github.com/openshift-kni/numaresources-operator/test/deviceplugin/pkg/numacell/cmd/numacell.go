/*
Copyright 2020.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"k8s.io/klog/v2"

	"github.com/ghodss/yaml"
	"github.com/jaypipes/ghw/pkg/option"
	"github.com/jaypipes/ghw/pkg/topology"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"

	"github.com/openshift-kni/numaresources-operator/test/deviceplugin/pkg/numacell/api"
	"github.com/openshift-kni/numaresources-operator/test/deviceplugin/pkg/numacell/manifests"
	"github.com/openshift-kni/numaresources-operator/test/deviceplugin/pkg/numacell/plugin"
)

func summarize(topoInfo *topology.Info) string {
	var buf strings.Builder
	for _, node := range topoInfo.Nodes {
		fmt.Fprintf(&buf, "NUMA node %d\n", node.ID)
		for _, core := range node.Cores {
			fmt.Fprintf(&buf, "\t%s\n", core.String())
		}
	}
	return buf.String()
}

func render(w io.Writer) int {
	nodeSelector := map[string]string{
		"${NODELABEL}": "${NODEVALUE}",
	}
	namespace := "${NAMESPACE}"
	name := "${NAME}"
	image := "${IMAGE}"
	sa := manifests.ServiceAccount(namespace, name)
	ro := manifests.Role(namespace, name)
	rb := manifests.RoleBinding(namespace, name)
	ds := manifests.DaemonSet(nodeSelector, namespace, name, sa.Name, image)
	for _, obj := range []interface{}{sa, ro, rb, ds} {
		data, err := yaml.Marshal(obj)
		if err != nil {
			return 1
		}
		fmt.Fprintf(w, "---\n%s", string(data))
	}
	return 0
}

func Execute() {
	var renderManifest bool
	var sysfsPath string
	var deviceCount int
	flag.BoolVar(&renderManifest, "render", false, "render daemonset manifest and exit")
	flag.StringVar(&sysfsPath, "sysfs", "/sys", "mount path of sysfs")
	flag.IntVar(&deviceCount, "devices", api.NUMACellDefaultDeviceCount, "amount of devices to expose (will not be decremented anyway)")
	flag.Parse()

	if renderManifest {
		os.Exit(render(os.Stdout))
	}

	klog.Infof("using sysfs at %q", sysfsPath)
	topoInfo, err := topology.New(option.WithPathOverrides(option.PathOverrides{
		"/sys": sysfsPath,
	}))
	if err != nil {
		klog.Fatalf("error getting topology info from %q: %v", sysfsPath, err)
	}

	klog.Infof("hardware detected:\n%s", summarize(topoInfo))

	manager := dpm.NewManager(plugin.NewNUMACellLister(topoInfo, deviceCount))
	manager.Run()
}
