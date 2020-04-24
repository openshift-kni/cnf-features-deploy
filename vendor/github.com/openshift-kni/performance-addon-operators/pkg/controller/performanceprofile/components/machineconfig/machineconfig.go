package machineconfig

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"

	"github.com/coreos/go-systemd/unit"
	igntypes "github.com/coreos/ignition/config/v2_2/types"

	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	profile2 "github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/profile"

	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	defaultIgnitionVersion       = "2.2.0"
	defaultFileSystem            = "root"
	defaultIgnitionContentSource = "data:text/plain;charset=utf-8;base64"
)

const (
	// MCKernelRT is the value of the kernel setting in MachineConfig for the RT kernel
	MCKernelRT = "realtime"
	// MCKernelDefault is the value of the kernel setting in MachineConfig for the default kernel
	MCKernelDefault = "default"

	preBootTuning       = "pre-boot-tuning"
	reboot              = "reboot"
	hugepagesAllocation = "hugepages-allocation"
	bashScriptsDir      = "/usr/local/bin"
)

const (
	systemdSectionUnit     = "Unit"
	systemdSectionService  = "Service"
	systemdSectionInstall  = "Install"
	systemdDescription     = "Description"
	systemdWants           = "Wants"
	systemdAfter           = "After"
	systemdBefore          = "Before"
	systemdEnvironment     = "Environment"
	systemdType            = "Type"
	systemdRemainAfterExit = "RemainAfterExit"
	systemdExecStart       = "ExecStart"
	systemdWantedBy        = "WantedBy"
)

const (
	systemdServiceKubelet      = "kubelet.service"
	systemdServiceTypeOneshot  = "oneshot"
	systemdTargetMultiUser     = "multi-user.target"
	systemdTargetNetworkOnline = "network-online.target"
	systemdTrue                = "true"
)

const (
	environmentHugepagesSize           = "HUGEPAGES_SIZE"
	environmentHugepagesCount          = "HUGEPAGES_COUNT"
	environmentNUMANode                = "NUMA_NODE"
	environmentReservedCpus            = "RESERVED_CPUS"
	environmentReservedCPUMaskInverted = "RESERVED_CPU_MASK_INVERT"
)

// New returns new machine configuration object for performance sensetive workflows
func New(assetsDir string, profile *performancev1alpha1.PerformanceProfile) (*machineconfigv1.MachineConfig, error) {
	name := components.GetComponentName(profile.Name, components.ComponentNamePrefix)
	mc := &machineconfigv1.MachineConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: machineconfigv1.GroupVersion.String(),
			Kind:       "MachineConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: profile2.GetMachineConfigLabel(profile),
		},
		Spec: machineconfigv1.MachineConfigSpec{},
	}

	ignitionConfig, err := getIgnitionConfig(assetsDir, profile)
	if err != nil {
		return nil, err
	}

	mc.Spec.Config = *ignitionConfig

	mc.Spec.KernelArguments, err = getKernelArgs(profile)

	if err != nil {
		return nil, err
	}

	enableRTKernel := profile.Spec.RealTimeKernel != nil &&
		profile.Spec.RealTimeKernel.Enabled != nil &&
		*profile.Spec.RealTimeKernel.Enabled

	if enableRTKernel {
		mc.Spec.KernelType = MCKernelRT
	} else {
		mc.Spec.KernelType = MCKernelDefault
	}

	return mc, nil
}

func getKernelArgs(profile *performancev1alpha1.PerformanceProfile) ([]string, error) {
	kargs := []string{
		"nohz=on",
		"nosoftlockup",
		"skew_tick=1",
		"intel_pstate=disable",
		"intel_iommu=on",
		"iommu=pt",
	}

	if profile.Spec.CPU.Isolated != nil {
		if profile.Spec.CPU.BalanceIsolated != nil && *profile.Spec.CPU.BalanceIsolated == false {
			kargs = append(kargs, fmt.Sprintf("isolcpus=%s", *profile.Spec.CPU.Isolated))
		}
		kargs = append(kargs, fmt.Sprintf("rcu_nocbs=%s", *profile.Spec.CPU.Isolated))
	}

	if profile.Spec.CPU.Reserved != nil {
		cpuMask, err := components.CPUListToMaskList(string(*profile.Spec.CPU.Reserved))
		if err != nil {
			return nil, err
		}
		kargs = append(kargs, fmt.Sprintf("tuned.non_isolcpus=%s", cpuMask))
	}

	if profile.Spec.HugePages != nil {
		if profile.Spec.HugePages.DefaultHugePagesSize != nil {
			kargs = append(kargs, fmt.Sprintf("default_hugepagesz=%s", *profile.Spec.HugePages.DefaultHugePagesSize))
		}

		for _, page := range profile.Spec.HugePages.Pages {
			// we can not allocate hugepages on the specific NUMA node via kernel boot arguments
			if page.Node != nil {
				continue
			}

			kargs = append(kargs, fmt.Sprintf("hugepagesz=%s", string(page.Size)))
			kargs = append(kargs, fmt.Sprintf("hugepages=%d", page.Count))
		}
	}

	if profile.Spec.AdditionalKernelArgs != nil {
		kargs = append(kargs, profile.Spec.AdditionalKernelArgs...)
	}

	return kargs, nil
}

func getIgnitionConfig(assetsDir string, profile *performancev1alpha1.PerformanceProfile) (*igntypes.Config, error) {

	mode := 0700
	ignitionConfig := &igntypes.Config{
		Ignition: igntypes.Ignition{
			Version: defaultIgnitionVersion,
		},
		Storage: igntypes.Storage{
			Files: []igntypes.File{},
		},
	}

	for _, script := range []string{preBootTuning, hugepagesAllocation, reboot} {
		content, err := ioutil.ReadFile(fmt.Sprintf("%s/scripts/%s.sh", assetsDir, script))
		if err != nil {
			return nil, err
		}
		contentBase64 := base64.StdEncoding.EncodeToString(content)
		ignitionConfig.Storage.Files = append(ignitionConfig.Storage.Files, igntypes.File{
			Node: igntypes.Node{
				Filesystem: defaultFileSystem,
				Path:       getBashScriptPath(script),
			},
			FileEmbedded1: igntypes.FileEmbedded1{
				Contents: igntypes.FileContents{
					Source: fmt.Sprintf("%s,%s", defaultIgnitionContentSource, contentBase64),
				},
				Mode: &mode,
			},
		})
	}

	if profile.Spec.CPU.Reserved != nil {
		reservedCpus := profile.Spec.CPU.Reserved
		contentBase64 := base64.StdEncoding.EncodeToString([]byte("[Manager]\nCPUAffinity=" + string(*reservedCpus)))
		ignitionConfig.Storage.Files = append(ignitionConfig.Storage.Files, igntypes.File{
			Node: igntypes.Node{
				Filesystem: defaultFileSystem,
				Path:       "/etc/systemd/system.conf.d/setAffinity.conf",
			},
			FileEmbedded1: igntypes.FileEmbedded1{
				Contents: igntypes.FileContents{
					Source: fmt.Sprintf("%s,%s", defaultIgnitionContentSource, contentBase64),
				},
				Mode: &mode,
			},
		})

		cpuInvertedMask, err := components.CPUListTo64BitsMaskList(string(*reservedCpus))
		if err != nil {
			return nil, err
		}

		preBootTuningService, err := getSystemdContent(
			getPreBootTuningUnitOptions(string(*reservedCpus), cpuInvertedMask),
		)
		if err != nil {
			return nil, err
		}

		rebootService, err := getSystemdContent(getRebootUnitOptions())
		if err != nil {
			return nil, err
		}

		ignitionConfig.Systemd = igntypes.Systemd{
			Units: []igntypes.Unit{
				{
					Contents: preBootTuningService,
					Enabled:  pointer.BoolPtr(true),
					Name:     getSystemdService(preBootTuning),
				},
				{
					Contents: rebootService,
					Enabled:  pointer.BoolPtr(true),
					Name:     getSystemdService(reboot),
				},
			},
		}
	}

	if profile.Spec.HugePages != nil {
		for _, page := range profile.Spec.HugePages.Pages {
			// we already allocated non NUMA specific hugepages via kernel arguments
			if page.Node == nil {
				continue
			}

			hugepagesSize, err := GetHugepagesSizeKilobytes(page.Size)
			if err != nil {
				return nil, err
			}

			hugepagesService, err := getSystemdContent(getHugepagesAllocationUnitOptions(
				hugepagesSize,
				page.Count,
				*page.Node,
			))
			if err != nil {
				return nil, err
			}

			ignitionConfig.Systemd.Units = append(ignitionConfig.Systemd.Units, igntypes.Unit{
				Contents: hugepagesService,
				Enabled:  pointer.BoolPtr(true),
				Name:     getSystemdService(fmt.Sprintf("%s-%skB-NUMA%d", hugepagesAllocation, hugepagesSize, *page.Node)),
			})
		}
	}

	return ignitionConfig, nil
}

func getBashScriptPath(scriptName string) string {
	return fmt.Sprintf("%s/%s.sh", bashScriptsDir, scriptName)
}

func getSystemdEnvironment(key string, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

func getSystemdService(serviceName string) string {
	return fmt.Sprintf("%s.service", serviceName)
}

func getSystemdContent(options []*unit.UnitOption) (string, error) {
	outReader := unit.Serialize(options)
	outBytes, err := ioutil.ReadAll(outReader)
	if err != nil {
		return "", err
	}
	return string(outBytes), nil
}

func getRebootUnitOptions() []*unit.UnitOption {
	return []*unit.UnitOption{
		// [Unit]
		// Description
		unit.NewUnitOption(systemdSectionUnit, systemdDescription, "Reboot initiated by pre-boot-tuning"),
		// Wants
		unit.NewUnitOption(systemdSectionUnit, systemdWants, systemdTargetNetworkOnline),
		// After
		unit.NewUnitOption(systemdSectionUnit, systemdAfter, systemdTargetNetworkOnline),
		// Before
		unit.NewUnitOption(systemdSectionUnit, systemdBefore, systemdServiceKubelet),
		// [Service]
		// Type
		unit.NewUnitOption(systemdSectionService, systemdType, systemdServiceTypeOneshot),
		// RemainAfterExit
		unit.NewUnitOption(systemdSectionService, systemdRemainAfterExit, systemdTrue),
		// ExecStart
		unit.NewUnitOption(systemdSectionService, systemdExecStart, getBashScriptPath(reboot)),
		// [Install]
		// WantedBy
		unit.NewUnitOption(systemdSectionInstall, systemdWantedBy, systemdTargetMultiUser),
	}
}

func getPreBootTuningUnitOptions(reservedCpus string, cpuInvertedMask string) []*unit.UnitOption {
	return []*unit.UnitOption{
		// [Unit]
		// Description
		unit.NewUnitOption(systemdSectionUnit, systemdDescription, "Preboot tuning patch"),
		// Before
		unit.NewUnitOption(systemdSectionUnit, systemdBefore, systemdServiceKubelet),
		unit.NewUnitOption(systemdSectionUnit, systemdBefore, getSystemdService(reboot)),
		// [Service]
		// Environment
		unit.NewUnitOption(systemdSectionService, systemdEnvironment, getSystemdEnvironment(environmentReservedCpus, reservedCpus)),
		unit.NewUnitOption(systemdSectionService, systemdEnvironment, getSystemdEnvironment(environmentReservedCPUMaskInverted, cpuInvertedMask)),
		// Type
		unit.NewUnitOption(systemdSectionService, systemdType, systemdServiceTypeOneshot),
		// RemainAfterExit
		unit.NewUnitOption(systemdSectionService, systemdRemainAfterExit, systemdTrue),
		// ExecStart
		unit.NewUnitOption(systemdSectionService, systemdExecStart, getBashScriptPath(preBootTuning)),
		// [Install]
		// WantedBy
		unit.NewUnitOption(systemdSectionInstall, systemdWantedBy, systemdTargetMultiUser),
	}
}

// GetHugepagesSizeKilobytes retruns hugepages size in kilobytes
func GetHugepagesSizeKilobytes(hugepagesSize performancev1alpha1.HugePageSize) (string, error) {
	switch hugepagesSize {
	case "1G":
		return "1048576", nil
	case "2M":
		return "2048", nil
	default:
		return "", fmt.Errorf("can not convert size %q to kilobytes", hugepagesSize)
	}
}

func getHugepagesAllocationUnitOptions(hugepagesSize string, hugepagesCount int32, numaNode int32) []*unit.UnitOption {
	return []*unit.UnitOption{
		// [Unit]
		// Description
		unit.NewUnitOption(systemdSectionUnit, systemdDescription, fmt.Sprintf("Hugepages-%skB allocation on the node %d", hugepagesSize, numaNode)),
		// Before
		unit.NewUnitOption(systemdSectionUnit, systemdBefore, systemdServiceKubelet),
		// [Service]
		// Environment
		unit.NewUnitOption(systemdSectionService, systemdEnvironment, getSystemdEnvironment(environmentHugepagesCount, fmt.Sprint(hugepagesCount))),
		unit.NewUnitOption(systemdSectionService, systemdEnvironment, getSystemdEnvironment(environmentHugepagesSize, hugepagesSize)),
		unit.NewUnitOption(systemdSectionService, systemdEnvironment, getSystemdEnvironment(environmentNUMANode, fmt.Sprint(numaNode))),
		// Type
		unit.NewUnitOption(systemdSectionService, systemdType, systemdServiceTypeOneshot),
		// RemainAfterExit
		unit.NewUnitOption(systemdSectionService, systemdRemainAfterExit, systemdTrue),
		// ExecStart
		unit.NewUnitOption(systemdSectionService, systemdExecStart, getBashScriptPath(hugepagesAllocation)),
		// [Install]
		// WantedBy
		unit.NewUnitOption(systemdSectionInstall, systemdWantedBy, systemdTargetMultiUser),
	}
}
