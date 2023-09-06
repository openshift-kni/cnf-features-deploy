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
	"fmt"
	"io"

	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

// WriteJSONEvent ...
func WriteJSONEvent(in *Event, writer io.Writer, stream *jsoniter.Stream) error {
	stream.WriteObjectStart()

	// Let's write the body
	if in != nil {
		var err error
		data := in
		if data.OdataContext != "" {
			stream.WriteObjectField("@odata.context")
			stream.WriteString(data.OdataContext)
			stream.WriteMore()
		}
		if data.Actions != nil {
			stream.WriteObjectField("Actions")
			_, err = stream.Write(data.Actions)
			if err != nil {
				return fmt.Errorf("error writing Actions: %w", err)
			}
			stream.WriteMore()
		}
		if data.Context != "" {
			stream.WriteObjectField("Context")
			stream.WriteString(data.Context)
			stream.WriteMore()
		}
		if data.Description != "" {
			stream.WriteObjectField("Description")
			stream.WriteString(data.Description)
			stream.WriteMore()
		}
		if data.Oem != nil {
			stream.WriteObjectField("Oem")
			_, err = stream.Write(data.Oem)
			if err != nil {
				return fmt.Errorf("error writing Oem: %w", err)
			}
		}
		if data.OdataType != "" {
			stream.WriteObjectField("@odata.type")
			stream.WriteString(data.OdataType)
			stream.WriteMore()
		} else {
			log.Warningf("@odata.type is not set")
		}
		if data.Events != nil {
			stream.WriteObjectField("Events")
			stream.WriteArrayStart()
			count := 0
			for i, v := range data.Events {
				if count > 0 {
					stream.WriteMore()
				}
				count++
				nv := v
				if err := writeJSONEventRecord(&nv, stream); err != nil {
					return fmt.Errorf("error writing Event[%d]: %w", i, err)
				}
			}
			stream.WriteArrayEnd()
			stream.WriteMore()
		} else {
			log.Warningf("field Events is not set")
		}
		if data.ID != "" {
			stream.WriteObjectField("Id")
			stream.WriteString(data.ID)
			stream.WriteMore()
		} else {
			log.Warningf("field Id is not set")
		}
		if data.Name != "" {
			stream.WriteObjectField("Name")
			stream.WriteString(data.Name)
		} else {
			return fmt.Errorf("field Name is not set")
		}

		stream.WriteObjectEnd()
	} else {
		return fmt.Errorf("Name is not set")
	}

	// Let's do a check on the error
	if stream.Error != nil {
		return fmt.Errorf("error while writing the RedfishEvent data: %w", stream.Error)
	}
	return nil
}

func writeJSONEventRecord(in *EventRecord, stream *jsoniter.Stream) error {
	stream.WriteObjectStart()

	// Let's write the body
	if in != nil {
		var err error
		data := in
		if data.Actions != nil {
			stream.WriteObjectField("Actions")
			_, err = stream.Write(data.Actions)
			if err != nil {
				return fmt.Errorf("error writing Actions: %w", err)
			}
			stream.WriteMore()
		}
		if data.Context != "" {
			stream.WriteObjectField("Context")
			stream.WriteString(data.Context)
			stream.WriteMore()
		}
		stream.WriteObjectField("EventGroupId")
		stream.WriteInt(data.EventGroupID)
		stream.WriteMore()
		if data.EventID != "" {
			stream.WriteObjectField("EventId")
			stream.WriteString(data.EventID)
			stream.WriteMore()
		}
		if data.EventTimestamp != "" {
			stream.WriteObjectField("EventTimestamp")
			stream.WriteString(data.EventTimestamp)
			stream.WriteMore()
		}
		if data.Message != "" {
			stream.WriteObjectField("Message")
			stream.WriteString(data.Message)
			stream.WriteMore()
		}
		if data.MessageArgs != nil {
			stream.WriteObjectField("MessageArgs")
			stream.WriteArrayStart()
			count := 0
			for _, v := range data.MessageArgs {
				if count > 0 {
					stream.WriteMore()
				}
				count++
				stream.WriteString(v)
			}
			stream.WriteArrayEnd()
			stream.WriteMore()
		}
		if data.Oem != nil {
			stream.WriteObjectField("Oem")
			_, err = stream.Write(data.Oem)
			if err != nil {
				return fmt.Errorf("error writing Oem: %w", err)
			}
			stream.WriteMore()
		}
		if data.OriginOfCondition != nil {
			stream.WriteObjectField("OriginOfCondition")
			_, err = stream.Write(data.OriginOfCondition)
			if err != nil {
				return fmt.Errorf("error writing OriginOfCondition: %w", err)
			}
			stream.WriteMore()
		}
		if data.Severity != "" {
			stream.WriteObjectField("Severity")
			stream.WriteString(data.Severity)
			stream.WriteMore()
		}
		if data.Resolution != "" {
			stream.WriteObjectField("Resolution")
			stream.WriteString(data.Resolution)
			stream.WriteMore()
		}
		if data.MessageID != "" {
			stream.WriteObjectField("MessageId")
			stream.WriteString(data.MessageID)
			stream.WriteMore()
		} else {
			log.Warningf("field MessageId is not set")
		}
		if data.MemberID != "" {
			stream.WriteObjectField("MemberId")
			stream.WriteString(data.MemberID)
			stream.WriteMore()
		} else {
			log.Warningf("field MemberId is not set")
		}
		if data.EventType != "" {
			stream.WriteObjectField("EventType")
			stream.WriteString(data.EventType)
		} else {
			return fmt.Errorf("field EventType is not set")
		}
		stream.WriteObjectEnd()
	} else {
		return fmt.Errorf("EventType is not set")
	}

	// Let's do a check on the error
	if stream.Error != nil {
		return fmt.Errorf("error while writing the EventRecord data: %w", stream.Error)
	}
	return nil
}
