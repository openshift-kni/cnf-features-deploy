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
	"net/url"
	"strings"
	"time"

	"github.com/redhat-cne/sdk-go/pkg/types"
)

var _ Writer = (*Event)(nil)

// SetType implements Writer.SetType
func (e *Event) SetType(t string) {
	e.Type = t
}

// SetSource implements Writer.SetSource
func (e *Event) SetSource(s string) {
	e.Source = s
}

// SetID implements Writer.SetID
func (e *Event) SetID(id string) {
	e.ID = id
}

// SetTime implements Writer.SetTime
func (e *Event) SetTime(t time.Time) {
	if t.IsZero() {
		e.Time = nil
	} else {
		e.Time = &types.Timestamp{Time: t}
	}
}

// SetDataSchema implements Writer.SetDataSchema
func (e *Event) SetDataSchema(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		e.DataSchema = nil
	}
	pu, err := url.Parse(s)
	if err != nil {
		return err
	}
	e.DataSchema = &types.URI{URL: *pu}
	return nil
}

// SetDataContentType implements Writer.SetDataContentType
func (e *Event) SetDataContentType(ct string) {
	ct = strings.TrimSpace(ct)
	if ct == "" {
		e.DataContentType = nil
	} else {
		e.DataContentType = &ct
	}
}

//SetData ...
func (e *Event) SetData(data Data) {
	nData := Data{
		Version: data.Version,
	}

	var nValues []DataValue

	for _, v := range data.Values {
		nValue := DataValue{
			Resource:  v.Resource,
			DataType:  v.DataType,
			ValueType: v.ValueType,
			Value:     v.Value,
		}
		nValues = append(nValues, nValue)
	}
	nData.Values = nValues
	e.Data = &nData
}
