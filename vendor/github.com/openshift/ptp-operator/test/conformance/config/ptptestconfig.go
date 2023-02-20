package test

import (
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/creasty/defaults"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type yamlTimeDur time.Duration

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
	DisableSoakTest  bool  `yaml:"disable_all"`
	FailureThreshold int   `default:"1"`
	Duration         int64 `yaml:"duration"`

	SlaveClockSyncConfig SlaveClockSync `yaml:"slave_clock_sync"`
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

func (conf *PtpTestConfig) loadPtpTestConfig(filename string) {
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, &conf)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
}

func GetPtpTestConfig() PtpTestConfig {
	if loaded {
		return ptpTestConfig
	}
	loaded = true

	// If config file path is provided, use that, otherwise continue with the provided config file
	path, ok := os.LookupEnv("PTP_TEST_CONFIG_FILE")
	if !ok {
		path = "../config/ptptestconfig.yaml"
	}

	ptpTestConfig.loadPtpTestConfig(path)
	logrus.Info("config=", ptpTestConfig)
	return ptpTestConfig
}
