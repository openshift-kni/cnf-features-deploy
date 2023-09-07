// Copyright 2021 The Cloud Native Events Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ptp

// SyncState ...
type SyncState string

const (
	// ACQUIRING_SYNC ...
	ACQUIRING_SYNC SyncState = "ACQUIRING-SYNC" // nolint:golint

	// ANTENNA_DISCONNECTED ...
	ANTENNA_DISCONNECTED SyncState = "ANTENNA-DISCONNECTED" // nolint:golint

	// ANTENNA_SHORT_CIRCUIT ...
	ANTENNA_SHORT_CIRCUIT SyncState = "ANTENNA-SHORT-CIRCUIT" // nolint:golint

	// BOOTING ...
	BOOTING SyncState = "BOOTING"

	// FREERUN ...
	FREERUN SyncState = "FREERUN"

	// HOLDOVER ...
	HOLDOVER SyncState = "HOLDOVER"

	// LOCKED ...
	LOCKED SyncState = "LOCKED"

	// SYNCHRONIZED ...
	SYNCHRONIZED SyncState = "SYNCHRONIZED"

	// UNLOCKED ...
	UNLOCKED SyncState = "UNLOCKED"
)
