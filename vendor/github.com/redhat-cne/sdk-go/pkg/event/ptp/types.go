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

// EventType ...
type EventType string

const (
	// GnssStateChange is Notification used to inform about gnss synchronization state change
	GnssStateChange EventType = "event.sync.gnss-status.gnss-state-change"

	// OsClockSyncStateChange is the object contains information related to a notification
	OsClockSyncStateChange EventType = "event.sync.sync-status.os-clock-sync-state-change"

	// PtpClockClassChange is Notification used to inform about ptp clock class changes.
	PtpClockClassChange EventType = "event.sync.ptp-status.ptp-clock-class-change"

	// PtpStateChange is Notification used to inform about ptp synchronization state change
	PtpStateChange EventType = "event.sync.ptp-status.ptp-state-change"

	// SynceClockQualityChange is Notification used to inform about changes in the clock quality of the primary SyncE signal advertised in ESMC packets
	SynceClockQualityChange EventType = "event.sync.synce-status.sync-clock-quality-change"

	// SynceStateChange is Notification used to inform about synce synchronization state change
	SynceStateChange EventType = "event.sync.sync-status.synce-state-change"

	// SynceStateChangeExtended is Notification used to inform about synce synchronization state change, enhanced state information
	SynceStateChangeExtended EventType = "event.sync.synce-status.synce-state-change-extended"

	// SyncStateChange is Notification used to inform about synchronization state change
	SyncStateChange EventType = "event.sync.sync-status.synchronization-state-change"
)
