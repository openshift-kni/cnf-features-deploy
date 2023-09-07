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
	"fmt"
	"regexp"
)

// DataType ...
type DataType string

const (
	// NOTIFICATION ...
	NOTIFICATION DataType = "notification"
	// METRIC ...
	METRIC DataType = "metric"
)

// ValueType ...
type ValueType string

const (
	// ENUMERATION ...
	ENUMERATION ValueType = "enumeration"
	// DECIMAL ...
	DECIMAL ValueType = "decimal64.3"
	// REDFISH_EVENT ...
	REDFISH_EVENT ValueType = "redfish-event" //nolint:all
)

// Data ... cloud native events data
// Data Json payload is as follows,
//{
//	"version": "v1.0",
//	"values": [{
//		"resource": "/sync/sync-status/sync-state",
//		"dataType": "notification",
//		"valueType": "enumeration",
//		"value": "ACQUIRING-SYNC"
//		}, {
//		"resource": "/sync/sync-status/sync-state",
//		"dataType": "metric",
//		"valueType": "decimal64.3",
//		"value": 100.3
//		}, {
//		"resource": "/redfish/v1/Systems",
//		"dataType": "notification",
//		"valueType": "redfish-event",
//		"value": {
// 		    "@odata.context": "/redfish/v1/$metadata#Event.Event",
// 		    "@odata.type": "#Event.v1_3_0.Event",
// 		    "Context": "any string is valid",
// 		    "Events": [{"EventId": "2162", "MemberId": "615703", "MessageId": "TMP0100"}],
// 		    "Id": "5e004f5a-e3d1-11eb-ae9c-3448edf18a38",
// 		    "Name": "Event Array"
//		}
//	}]
//}
type Data struct {
	Version string      `json:"version" example:"v1"`
	Values  []DataValue `json:"values"`
}

// DataValue ...
// DataValue Json payload is as follows,
//{
//	"resource": "/cluster/node/ptp",
//	"dataType": "notification",
//	"valueType": "enumeration",
//	"value": "ACQUIRING-SYNC"
//}
type DataValue struct {
	Resource  string      `json:"resource" example:"/cluster/node/clock"`
	DataType  DataType    `json:"dataType" example:"metric"`
	ValueType ValueType   `json:"valueType" example:"decimal64.3"`
	Value     interface{} `json:"value" example:"100.3"`
}

// SetVersion  ...
func (d *Data) SetVersion(s string) error {
	d.Version = s
	if s == "" {
		err := fmt.Errorf("version cannot be empty")
		return err
	}
	return nil
}

// SetValues ...
func (d *Data) SetValues(v []DataValue) {
	d.Values = v
}

// AppendValues ...
func (d *Data) AppendValues(v DataValue) {
	d.Values = append(d.Values, v)
}

// GetVersion ...
func (d *Data) GetVersion() string {
	return d.Version
}

// GetValues ...
func (d *Data) GetValues() []DataValue {
	return d.Values
}

// GetResource ...
func (v *DataValue) GetResource() string {
	return v.Resource
}

// SetResource ...
func (v *DataValue) SetResource(r string) error {
	matched, err := regexp.MatchString(`([^/]+(/{2,}[^/]+)?)`, r)
	if matched {
		v.Resource = r
	} else {
		return err
	}
	return nil
}
