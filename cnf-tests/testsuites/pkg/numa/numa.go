package numa

import (
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/strings/slices"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/pods"
)

// FindNUMAForCPUs finds the NUMA node if all the CPUs in the list are in the same one and returns it.
func FindForCPUs(pod *corev1.Pod, cpuList []string) (int, error) {
	buff, err := pods.ExecCommand(client.Client, *pod, []string{"lscpu", "--parse=cpu,node"})
	if err != nil {
		return -1, fmt.Errorf("cannot issue lscpu on pod %s/%s: %w", pod.Namespace, pod.Name, err)

	}

	return findForCPUsParseOutput(buff.String(), cpuList)
}

func findForCPUsParseOutput(lscpuOutput string, cpuList []string) (int, error) {
	foundNUMAnode := -1
	separator := "\r\n"
	if !strings.Contains(lscpuOutput, separator) {
		separator = "\n"
	}

	for _, line := range strings.Split(lscpuOutput, separator) {
		if strings.HasPrefix(line, "#") {
			continue
		}

		if !strings.Contains(line, ",") {
			// Last line is empty
			continue
		}

		splittedLine := strings.Split(line, ",")
		if len(splittedLine) != 2 {
			return -1, fmt.Errorf("bad line output for lscpu: %s", line)
		}

		cpu := splittedLine[0]
		numa, err := strconv.Atoi(splittedLine[1])
		if err != nil {
			return -1, fmt.Errorf("can't convert NUMA node from line: %s. lscpu output: %s", line, lscpuOutput)
		}

		if slices.Contains(cpuList, cpu) {
			if foundNUMAnode != -1 {
				if foundNUMAnode != numa {
					return -1, fmt.Errorf("not all the cpus are in the same numa node")
				}
			}

			foundNUMAnode = numa
		}
	}

	return foundNUMAnode, nil
}
