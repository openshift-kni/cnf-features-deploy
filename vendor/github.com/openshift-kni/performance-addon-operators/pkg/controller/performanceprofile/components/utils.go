package components

import (
	"bytes"
	"fmt"
	"math/big"
	"strings"

	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
)

const maxSystemCpus = 64

// GetComponentName returns the component name for the specific performance profile
func GetComponentName(profileName string, prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, profileName)
}

// GetFirstKeyAndValue return the first key / value pair of a map
func GetFirstKeyAndValue(m map[string]string) (string, string) {
	for k, v := range m {
		return k, v
	}
	return "", ""
}

// SplitLabelKey returns the given label key splitted up in domain and role
func SplitLabelKey(s string) (domain, role string, err error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("Can't split %s", s)
	}
	return parts[0], parts[1], nil
}

// CPUListToHexMask converts a list of cpus into a cpu mask represented in hexdecimal
func CPUListToHexMask(cpulist string) (hexMask string, err error) {
	cpus, err := cpuset.Parse(cpulist)
	if err != nil {
		return "", err
	}

	reservedCPUs := cpus.ToSlice()
	currMask := big.NewInt(0)
	for _, cpu := range reservedCPUs {
		x := new(big.Int).Lsh(big.NewInt(1), uint(cpu))
		currMask.Or(currMask, x)
	}
	return fmt.Sprintf("%064x", currMask), nil
}

// CPUListToInvertedMask converts a list of cpus into an inverted cpu mask represented in hexdecimal
func CPUListToInvertedMask(cpulist string) (hexMask string, err error) {
	cpus, err := cpuset.Parse(cpulist)
	if err != nil {
		return "", err
	}

	reservedCPUs := cpus.ToSlice()

	reservedCpusLookup := make(map[int]bool)
	for _, cpu := range reservedCPUs {
		reservedCpusLookup[cpu] = true
	}

	currMask := big.NewInt(0)
	for cpu := 0; cpu < maxSystemCpus; cpu++ {
		if _, reserved := reservedCpusLookup[cpu]; reserved {
			continue
		}
		x := new(big.Int).Lsh(big.NewInt(1), uint(cpu))
		currMask.Or(currMask, x)
	}
	return fmt.Sprintf("%016x", currMask), nil
}

// CPUListTo64BitsMaskList converts a list of cpus into an inverted cpu mask represented
// in a list of 64bit hexadecimal mask devided by a delimiter ","
func CPUListTo64BitsMaskList(cpulist string) (hexMask string, err error) {
	maskStr, err := CPUListToInvertedMask(cpulist)
	if err != nil {
		return "", nil
	}
	return fmt.Sprintf("%s,%s", maskStr[:8], maskStr[8:]), nil
}

// CPUListToMaskList converts a list of cpus into a cpu mask represented
// in a list of hexadecimal mask devided by a delimiter ","
func CPUListToMaskList(cpulist string) (hexMask string, err error) {
	maskStr, err := CPUListToHexMask(cpulist)
	if err != nil {
		return "", nil
	}
	index := 0
	for index < (len(maskStr) - 8) {
		if maskStr[index:index+8] != "00000000" {
			break
		}
		index = index + 8
	}
	var b bytes.Buffer
	for index <= (len(maskStr) - 16) {
		b.WriteString(maskStr[index : index+8])
		b.WriteString(",")
		index = index + 8
	}
	b.WriteString(maskStr[index : index+8])
	trimmedCPUMaskList := b.String()
	return trimmedCPUMaskList, nil
}

// CPUListIntersect returns cpu ids found in both the provided cpuLists, if any
func CPUListIntersect(cpuListA, cpuListB string) ([]int, error) {
	var err error
	cpusA, err := cpuset.Parse(cpuListA)
	if err != nil {
		return nil, err
	}
	cpusB, err := cpuset.Parse(cpuListB)
	if err != nil {
		return nil, err
	}
	commonSet := cpusA.Intersection(cpusB)
	return commonSet.ToSlice(), nil
}
