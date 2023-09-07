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

package types

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"time"
)

// Timestamp wraps time.Time to normalize the time layout to RFC3339. It is
// intended to enforce compliance with the Cloud Native events spec for their
// definition of Timestamp. Custom marshal methods are implemented to ensure
// the outbound Timestamp is a string in the RFC3339 layout.
type Timestamp struct {
	time.Time
}

// ParseTimestamp attempts to parse the given time assuming RFC3339 layout
func ParseTimestamp(s string) (*Timestamp, error) {
	if s == "" {
		return nil, nil
	}
	tt, err := ParseTime(s)
	return &Timestamp{Time: tt}, err
}

// MarshalJSON implements a custom json marshal method used when this type is
// marshaled using json.Marshal.
func (t *Timestamp) MarshalJSON() ([]byte, error) {
	if t == nil || t.IsZero() {
		return []byte(`""`), nil
	}
	return []byte(fmt.Sprintf("%q", t)), nil
}

// UnmarshalJSON implements the json unmarshal method used when this type is
// unmarshalled using json.Unmarshal.
func (t *Timestamp) UnmarshalJSON(b []byte) error {
	var timestamp string
	if err := json.Unmarshal(b, &timestamp); err != nil {
		return err
	}
	var err error
	t.Time, err = ParseTime(timestamp)
	return err
}

// MarshalXML implements a custom xml marshal method used when this type is
// marshaled using xml.Marshal.
func (t *Timestamp) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if t == nil || t.IsZero() {
		return e.EncodeElement(nil, start)
	}
	return e.EncodeElement(t.String(), start)
}

// UnmarshalXML implements the xml unmarshal method used when this type is
// unmarshaled using xml.Unmarshal.
func (t *Timestamp) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var timestamp string
	if err := d.DecodeElement(&timestamp, &start); err != nil {
		return err
	}
	var err error
	t.Time, err = ParseTime(timestamp)
	return err
}

// String outputs the time using RFC3339 format.
func (t Timestamp) String() string { return FormatTime(t.Time) }
