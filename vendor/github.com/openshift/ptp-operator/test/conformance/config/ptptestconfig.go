package test

import (
	"fmt"
	"os"

	"github.com/creasty/defaults"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	// Default cpu usage threshold in milliCpus.
	PtpDefaultMilliCoresUsageThreshold = 15
)

type GlobalConfig struct {
	MinOffset int `yaml:"minoffset"`
	MaxOffset int `yaml:"maxoffset"`
}

type TestSpec struct {
	Enable           bool  `default:"true"`
	FailureThreshold int   `yaml:"failure_threshold"`
	Duration         int64 `yaml:"duration"`
}

func (t *TestSpec) UnmarshalYAML(unmarshal func(interface{}) error) error {
	defaults.Set(t)

	type plain TestSpec
	if err := unmarshal((*plain)(t)); err != nil {
		return err
	}
	return nil
}

type SoakTestConfig struct {
	DisableSoakTest      bool           `yaml:"disable_all"`
	FailureThreshold     int            `default:"1"`
	Duration             int64          `yaml:"duration"`
	EventOutputFile      string         `yaml:"event_output_file" default:"./event-output.csv"`
	SlaveClockSyncConfig SlaveClockSync `yaml:"slave_clock_sync"`
	CpuUtilization       CpuUtilization `yaml:"cpu_utilization"`
}

func (t *SoakTestConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	defaults.Set(t)

	type plain SoakTestConfig
	if err := unmarshal((*plain)(t)); err != nil {
		return err
	}
	return nil
}

type PtpTestConfig struct {
	GlobalConfig   GlobalConfig   `yaml:"global"`
	SoakTestConfig SoakTestConfig `yaml:"soaktest"`
}

// Individual test configuration

type SlaveClockSync struct {
	TestSpec    TestSpec `yaml:"spec"`
	Description string   `yaml:"desc"`
}

var ptpTestConfig PtpTestConfig
var loaded bool

func (conf *PtpTestConfig) loadPtpTestConfig(filename string) error {
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read ptp test config file %s: %v", filename, err)
	}

	err = yaml.Unmarshal(yamlFile, &conf)
	if err != nil {
		return fmt.Errorf("failed to unmarshal file %s: %v", filename, err)
	}
	return nil
}

func GetPtpTestConfig() (*PtpTestConfig, error) {
	if loaded {
		return &ptpTestConfig, nil
	}

	// If config file path is provided, use that, otherwise continue with the provided config file
	path, ok := os.LookupEnv("PTP_TEST_CONFIG_FILE")
	if !ok {
		path = "../config/ptptestconfig.yaml"
	}

	err := ptpTestConfig.loadPtpTestConfig(path)
	if err != nil {
		return nil, err
	}

	logrus.Infof("PTP Test Config: %+v", ptpTestConfig)
	loaded = true
	return &ptpTestConfig, nil
}
