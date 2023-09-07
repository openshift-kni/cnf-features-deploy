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

package redfish

import (
	jsoniter "github.com/json-iterator/go"
)

func readEventRecord(iter *jsoniter.Iterator) []EventRecord {
	var result []EventRecord
	for iter.ReadArray() {
		e := EventRecord{}
		for eField := iter.ReadObject(); eField != ""; eField = iter.ReadObject() {
			switch eField {
			case "Actions":
				e.Actions = iter.SkipAndReturnBytes()
			case "Context":
				e.Context = iter.ReadString()
			case "EventGroupId":
				e.EventGroupID = iter.ReadInt()
			case "EventId":
				e.EventID = iter.ReadString()
			case "EventTimestamp":
				e.EventTimestamp = iter.ReadString()
			case "EventType":
				e.EventType = iter.ReadString()
			case "MemberId":
				e.MemberID = iter.ReadString()
			case "Message":
				e.Message = iter.ReadString()
			case "MessageArgs":
				for iter.ReadArray() {
					arg := iter.ReadString()
					e.MessageArgs = append(e.MessageArgs, arg)
				}
			case "MessageId":
				e.MessageID = iter.ReadString()
			case "Oem":
				e.Oem = iter.SkipAndReturnBytes()
			case "OriginOfCondition":
				e.OriginOfCondition = iter.SkipAndReturnBytes()
			case "Severity":
				e.Severity = iter.ReadString()
			case "Resolution":
				e.Resolution = iter.ReadString()
			default:
				iter.Skip()
			}
		}
		result = append(result, e)
	}
	return result
}

// readJSONFromIterator allows you to read the bytes reader as an event
func readJSONFromIterator(out *Event, iter *jsoniter.Iterator) error {
	for key := iter.ReadObject(); key != ""; key = iter.ReadObject() {
		// Check if we have some error in our error cache
		if iter.Error != nil {
			return iter.Error
		}

		switch key {
		case "@odata.context":
			out.OdataContext = iter.ReadString()
		case "@odata.type":
			out.OdataType = iter.ReadString()
		case "Actions":
			out.Actions = iter.SkipAndReturnBytes()
		case "Context":
			out.Context = iter.ReadString()
		case "Description":
			out.Description = iter.ReadString()
		case "Id":
			out.ID = iter.ReadString()
		case "Name":
			out.Name = iter.ReadString()
		case "Oem":
			out.Oem = iter.SkipAndReturnBytes()
		case "Events":
			e := readEventRecord(iter)
			out.Events = e
		default:
			iter.Skip()
		}
	}
	return nil
}

// UnmarshalJSON implements the json unmarshal method used when this type is
// unmarshaled using json.Unmarshal.
func (e *Event) UnmarshalJSON(b []byte) error {
	iterator := jsoniter.ConfigFastest.BorrowIterator(b)
	defer jsoniter.ConfigFastest.ReturnIterator(iterator)
	return readJSONFromIterator(e, iterator)
}
