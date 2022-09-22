/*
Copyright 2021 The Kubernetes Authors.

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

package configuration

import (
	"fmt"
	"os"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/k8stopologyawareschedwg/deployer/pkg/deployer/platform"
	"github.com/k8stopologyawareschedwg/deployer/pkg/deployer/platform/detect"
)

const (
	envVarMCPUpdateTimeout  = "E2E_NROP_MCP_UPDATE_TIMEOUT"
	envVarMCPUpdateInterval = "E2E_NROP_MCP_UPDATE_INTERVAL"
	envVarPlatform          = "E2E_NROP_PLATFORM"
	envVarPlatformVersion   = "E2E_NROP_PLATFORM_VERSION"
)

const (
	defaultMCPUpdateTimeout  = 30 * time.Minute
	defaultMCPUpdateInterval = 30 * time.Second
	defaultOCPVersion        = "v4.11"
)

var (
	Plat                            platform.Platform
	PlatVersion                     platform.Version
	MachineConfigPoolUpdateTimeout  time.Duration
	MachineConfigPoolUpdateInterval time.Duration
)

func init() {
	var err error

	MachineConfigPoolUpdateTimeout, err = getMachineConfigPoolUpdateValueFromEnv(envVarMCPUpdateTimeout, defaultMCPUpdateTimeout)
	if err != nil {
		panic(fmt.Errorf("failed to parse machine config pool update timeout: %w", err))
	}

	MachineConfigPoolUpdateInterval, err = getMachineConfigPoolUpdateValueFromEnv(envVarMCPUpdateInterval, defaultMCPUpdateInterval)
	if err != nil {
		panic(fmt.Errorf("failed to parse machine config pool update interval: %w", err))
	}

	Plat, err = detect.Platform()
	if err != nil {
		Plat = getPlatformFromEnv(envVarPlatform)
	}
	if Plat == platform.Unknown {
		Plat = platform.OpenShift
		klog.Infof("forced to %q: failed to detect a platform: %w", Plat, err)
	}

	PlatVersion, err = detect.Version(Plat)
	if err != nil {
		PlatVersion, err = getPlatformVersionFromEnv(envVarPlatform)
		if err != nil {
			PlatVersion = platform.Version(defaultOCPVersion)
			klog.Infof("forced to %q: failed to detect platform version: %w", PlatVersion, err)
		}
	}
	if PlatVersion == platform.MissingVersion {
		PlatVersion = platform.Version(defaultOCPVersion)
		klog.Infof("forced to %q: failed to detect a platform: %w", Plat, err)
	}
}

func getPlatformFromEnv(envVar string) platform.Platform {
	val, ok := os.LookupEnv(envVar)
	if !ok {
		return platform.Unknown
	}
	switch strings.ToLower(val) {
	case "kubernetes":
		return platform.Kubernetes
	case "openshift":
		return platform.OpenShift
	}
	return platform.Unknown
}

func getPlatformVersionFromEnv(envVar string) (platform.Version, error) {
	val, ok := os.LookupEnv(envVar)
	if !ok {
		return platform.MissingVersion, fmt.Errorf("failed to get platform version from environment variable %s", envVar)
	}
	version, err := platform.ParseVersion(val)
	if err != nil {
		return platform.MissingVersion, fmt.Errorf("failed to parse platform version %s: %w", val, err)
	}
	return version, nil
}

func getMachineConfigPoolUpdateValueFromEnv(envVar string, fallback time.Duration) (time.Duration, error) {
	val, ok := os.LookupEnv(envVar)
	if !ok {
		return fallback, nil
	}
	return time.ParseDuration(val)
}
