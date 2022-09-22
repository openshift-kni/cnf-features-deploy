/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2022 Red Hat, Inc.
 */

package detect

import (
	"github.com/k8stopologyawareschedwg/deployer/pkg/deployer/platform"
)

type PlatformInfo struct {
	AutoDetected platform.Platform `json:"autoDetected"`
	UserSupplied platform.Platform `json:"userSupplied"`
	Discovered   platform.Platform `json:"discovered"`
}

type VersionInfo struct {
	AutoDetected platform.Version `json:"autoDetected"`
	UserSupplied platform.Version `json:"userSupplied"`
	Discovered   platform.Version `json:"discovered"`
}

type ClusterInfo struct {
	Platform PlatformInfo `json:"platform"`
	Version  VersionInfo  `json:"version"`
}

const (
	DetectedFromUser    string = "user-supplied"
	DetectedFromCluster string = "autodetected from cluster"
	DetectedFailure     string = "autodetection failed"
)

func FindPlatform(userSupplied platform.Platform) (PlatformInfo, string, error) {
	do := PlatformInfo{
		AutoDetected: platform.Unknown,
		UserSupplied: userSupplied,
		Discovered:   platform.Unknown,
	}

	if do.UserSupplied != platform.Unknown {
		do.Discovered = do.UserSupplied
		return do, DetectedFromUser, nil
	}

	dp, err := Platform()
	if err != nil {
		return do, DetectedFailure, err
	}

	do.AutoDetected = dp
	do.Discovered = do.AutoDetected
	return do, DetectedFromCluster, nil
}

func FindVersion(plat platform.Platform, userSupplied platform.Version) (VersionInfo, string, error) {
	do := VersionInfo{
		AutoDetected: platform.MissingVersion,
		UserSupplied: userSupplied,
		Discovered:   platform.MissingVersion,
	}

	if do.UserSupplied != platform.MissingVersion {
		do.Discovered = do.UserSupplied
		return do, DetectedFromUser, nil
	}

	dv, err := Version(plat)
	if err != nil {
		return do, DetectedFailure, err
	}

	do.AutoDetected = dv
	do.Discovered = do.AutoDetected
	return do, DetectedFromCluster, nil
}
