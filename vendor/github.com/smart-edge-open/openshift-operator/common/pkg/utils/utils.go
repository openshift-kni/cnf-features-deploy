// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type AcceleratorDiscoveryConfig struct {
	VendorID  map[string]string
	Class     string
	SubClass  string
	Devices   map[string]string
	NodeLabel string
}

const (
	configFilesizeLimitInBytes = 10485760 //10 MB
)

func LoadDiscoveryConfig(cfgPath string) (AcceleratorDiscoveryConfig, error) {
	var cfg AcceleratorDiscoveryConfig
	file, err := os.Open(filepath.Clean(cfgPath))
	if err != nil {
		return cfg, fmt.Errorf("Failed to open config: %v", err)
	}
	defer file.Close()

	// get file stat
	stat, err := file.Stat()
	if err != nil {
		return cfg, fmt.Errorf("Failed to get file stat: %v", err)
	}

	// check file size
	if stat.Size() > configFilesizeLimitInBytes {
		return cfg, fmt.Errorf("Config file size %d, exceeds limit %d bytes",
			stat.Size(), configFilesizeLimitInBytes)
	}

	cfgData := make([]byte, stat.Size())
	bytesRead, err := file.Read(cfgData)
	if err != nil || int64(bytesRead) != stat.Size() {
		return cfg, fmt.Errorf("Unable to read config: %s", filepath.Clean(cfgPath))
	}

	if err = json.Unmarshal(cfgData, &cfg); err != nil {
		return cfg, fmt.Errorf("Failed to unmarshal config: %v", err)
	}
	return cfg, nil
}
