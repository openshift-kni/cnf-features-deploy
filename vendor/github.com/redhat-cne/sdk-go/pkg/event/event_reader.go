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

var _ Reader = (*Event)(nil)

// GetType implements Reader.Type
func (e *Event) GetType() string {
	return e.Type
}

// GetSource implements Reader.Source
func (e *Event) GetSource() string {
	return e.Source
}

// GetID implements Reader.ID
func (e *Event) GetID() string {
	return e.ID
}

// GetTime implements Reader.Time
func (e *Event) GetTime() time.Time {
	if e.Time != nil {
		return e.Time.Time
	}
	return time.Time{}
}

// GetDataSchema implements Reader.DataSchema
func (e *Event) GetDataSchema() string {
	return e.DataSchema.String()
}

// GetDataContentType implements Reader.DataContentType
func (e *Event) GetDataContentType() string {
	return *e.DataContentType
}

// GetData ...
func (e *Event) GetData() *Data {
	return e.Data
}
