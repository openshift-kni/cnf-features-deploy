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
)

// SetDataValue encodes the given payload
func (e *Event) SetDataValue(dataType DataType, obj interface{}) (err error) {
	var data DataValue
	switch dataType {
	case NOTIFICATION:
		data.DataType = dataType
		data.ValueType = ENUMERATION
		data.Value = obj
	case METRIC:
		data.DataType = dataType
		data.ValueType = DECIMAL
		data.Value = obj
	default:
		err = fmt.Errorf("error setting Data %s - %v", dataType, obj)
	}
	e.Data.Values = append(e.Data.Values, data)
	return
}

// GetDataValue encodes the given payload
func (e *Event) GetDataValue() (data []DataValue, err error) {
	return e.Data.Values, nil
}
