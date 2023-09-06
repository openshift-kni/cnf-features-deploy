// Copyright 2020 The Cloud Native Events Authors
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

package event

import (
	"time"
)

// Reader is the interface for reading through an event from attributes.
type Reader interface {
	// GetType returns event.GetType().
	GetType() string
	// GetTime returns event.GetTime().
	GetTime() time.Time
	// GetID returns event.GetID().
	GetID() string
	// GetDataSchema returns event.GetDataSchema().
	GetDataSchema() string
	// GetDataContentType returns event.GetDataContentType().
	GetDataContentType() string
	// GetData returns event.GetData()
	GetData() *Data
	// Clone clones the event .
	Clone() Event
	// String returns a pretty-printed representation of the EventContext.
	String() string
}

// Writer is the interface for writing through an event onto attributes.
// If an error is thrown by a sub-component, Writer caches the error
// internally and exposes errors with a call to event.Validate().
type Writer interface {
	// SetType performs event.SetType.
	SetType(string)
	// SetID performs event.SetID.
	SetID(string)
	// SetTime performs event.SetTime.
	SetTime(time.Time)
	// SetDataSchema performs event.SetDataSchema.
	SetDataSchema(string) error
	// SetDataContentType performs event.SetDataContentType.
	SetDataContentType(string)
	// SetData
	SetData(Data)
}
