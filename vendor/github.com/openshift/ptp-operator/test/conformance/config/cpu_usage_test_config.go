package test

import (
	"fmt"
	"strings"
	"time"

	"github.com/creasty/defaults"
)

type CpuUsageNodeConfig struct {
	CpuUsageThreshold int `yaml:"cpu_threshold_mcores"`
}

type ContainerName string

type CpuUsagePodConfig struct {
	PodType           string         `yaml:"pod_type"`
	Container         *ContainerName `yaml:"container,omitempty"`
	CpuUsageThreshold int            `yaml:"cpu_threshold_mcores"`
}

type CpuUsagePodConfigs []CpuUsagePodConfig

type CpuUsageCustomParams struct {
	PromTimeWindow string              `yaml:"prometheus_rate_time_window"`
	Node           *CpuUsageNodeConfig `yaml:"node,omitempty"`
	Pod            *CpuUsagePodConfigs `yaml:"pod,omitempty"`
}

type CpuTestSpec struct {
	TestSpec
	CustomParams CpuUsageCustomParams `yaml:"custom_params"`
}

type CpuUtilization struct {
	CpuTestSpec CpuTestSpec `yaml:"spec"`
	Description string      `yaml:"desc"`
}

func (cfg *CpuUsageNodeConfig) String() string {
	return fmt.Sprintf("%+v", *cfg)
}

func (p *CpuUsagePodConfigs) String() string {
	return fmt.Sprintf("%+v", *p)
}

func (n *ContainerName) String() string {
	return string(*n)
}

func (t *CpuTestSpec) UnmarshalYAML(unmarshal func(interface{}) error) error {
	defaults.Set(&t.TestSpec)

	// Unmarshal base TestSpec fields
	type plain1 TestSpec
	if err := unmarshal((*plain1)(&t.TestSpec)); err != nil {
		return err
	}

	// Unmarshal custom test case params. We need a (temp) holder type.
	type CustomParamsHolder struct {
		Params CpuUsageCustomParams `yaml:"custom_params"`
	}
	customParams := CustomParamsHolder{}
	if err := unmarshal((*CustomParamsHolder)(&customParams)); err != nil {
		return err
	}

	t.CustomParams = customParams.Params
	return nil
}

func (config *CpuUtilization) PromRateTimeWindow() (time.Duration, error) {
	return time.ParseDuration(config.CpuTestSpec.CustomParams.PromTimeWindow)
}

func (config *CpuUtilization) ShouldCheckNodeTotalCpuUsage() (bool, float64) {
	if config.CpuTestSpec.CustomParams.Node == nil {
		return false, 0
	}

	return true, float64(config.CpuTestSpec.CustomParams.Node.CpuUsageThreshold) / 1000
}

func (config *CpuUtilization) ShouldCheckContainerCpuUsage(podName, containerName string) (bool, float64) {
	if config.CpuTestSpec.CustomParams.Pod == nil {
		return false, 0
	}

	for _, pod := range *config.CpuTestSpec.CustomParams.Pod {
		if !strings.Contains(podName, pod.PodType) || pod.Container == nil {
			continue
		}

		if string(*pod.Container) == containerName {
			return true, float64(pod.CpuUsageThreshold) / 1000
		}
	}

	// Container not found
	return false, 0
}

func (config *CpuUtilization) ShouldCheckPodCpuUsage(podName string) (bool, float64) {
	if config.CpuTestSpec.CustomParams.Pod == nil {
		return false, 0
	}

	for _, pod := range *config.CpuTestSpec.CustomParams.Pod {
		if strings.Contains(podName, pod.PodType) && pod.Container == nil {
			return true, float64(pod.CpuUsageThreshold) / 1000
		}
	}

	return false, 0
}
