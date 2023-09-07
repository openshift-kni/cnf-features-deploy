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
	"strconv"
	"strings"
)

// The structs defined here are based on Redfish Schema Bundle 2019.2.

// EventRecord is defined in Redfish Event_v1_4_1_EventRecord
// https://redfish.dmtf.org/schemas/v1/Event.v1_4_1.json
// Additional information returned from Message Registry is added
// from https://redfish.dmtf.org/schemas/v1/Message.v1_0_8.json
// Required fields: EventType, MessageId, MemberId
type EventRecord struct {
	// The Actions property shall contain the available actions
	// for this resource.
	// +optional
	Actions []byte `json:"Actions,omitempty"`
	// *deprecated* This property has been Deprecated in favor of Context
	// found at the root level of the object.
	// +optional
	Context string `json:"Context,omitempty"`
	// This value is the identifier used to correlate events that
	// came from the same cause.
	// +optional
	EventGroupID int `json:"EventGroupId,omitempty"`
	// The value of this property shall indicate a unique identifier
	// for the event, the format of which is implementation dependent.
	// +optional
	EventID string `json:"EventId,omitempty"`
	// This is time the event occurred.
	// +optional
	EventTimestamp string `json:"EventTimestamp,omitempty"`
	// *deprecated* This property has been deprecated.  Starting Redfish
	// Spec 1.6 (Event 1.3), subscriptions are based on RegistryId and ResourceType
	// and not EventType.
	// This indicates the type of event sent, according to the definitions
	// in the EventService.
	// +required
	EventType string `json:"EventType"`
	// This is the identifier for the member within the collection.
	// +required
	MemberID string `json:"MemberId"`
	// This property shall contain an optional human readable
	// message.
	// +optional
	Message string `json:"Message,omitempty"`
	// This array of message arguments are substituted for the arguments
	// in the message when looked up in the message registry.
	// +optional
	MessageArgs []string `json:"MessageArgs,omitempty"`
	// This property shall be a key into message registry as
	// described in the Redfish specification.
	// +required
	MessageID string `json:"MessageId"`
	// This is the manufacturer/provider specific extension
	// +optional
	Oem []byte `json:"Oem,omitempty"`
	// This indicates the resource that originated the condition that
	// caused the event to be generated.
	// +optional
	OriginOfCondition []byte `json:"OriginOfCondition,omitempty"`
	//  This is the severity of the event.
	// +optional
	Severity string `json:"Severity,omitempty"`
	// The following fields are defined in schema Message.v1_0_8
	// These are additional information returned from Message Registry.
	// Used to provide suggestions on how to resolve the situation that caused the error.
	// +optional
	Resolution string `json:"Resolution"`
}

// String returns a pretty-printed representation of the EventRecord.
func (e EventRecord) String() string {
	b := strings.Builder{}
	b.WriteString("\n")
	if e.Actions != nil {
		b.WriteString("      Actions: " + string(e.Actions) + "\n")
	}
	if e.Context != "" {
		b.WriteString("      Context: " + e.Context + "\n")
	}
	// EventGroupId shows 0 by default
	b.WriteString("      EventGroupId: " + strconv.Itoa(e.EventGroupID) + "\n")
	if e.EventID != "" {
		b.WriteString("      EventId: " + e.EventID + "\n")
	}
	if e.EventTimestamp != "" {
		b.WriteString("      EventTimestamp: " + e.EventTimestamp + "\n")
	}
	b.WriteString("      EventType: " + e.EventType + "\n")
	b.WriteString("      MemberId: " + e.MemberID + "\n")
	if e.Message != "" {
		b.WriteString("      Message: " + e.Message + "\n")
	}
	if e.MessageArgs != nil {
		b.WriteString("      MessageArgs: ")
		for _, arg := range e.MessageArgs {
			b.WriteString(arg + ", ")
		}
		b.WriteString("\n")
	}
	b.WriteString("      MessageId: " + e.MessageID + "\n")
	if e.Oem != nil {
		b.WriteString("      Oem: " + string(e.Oem) + "\n")
	}
	if e.OriginOfCondition != nil {
		b.WriteString("      OriginOfCondition: " + string(e.OriginOfCondition) + "\n")
	}
	if e.Severity != "" {
		b.WriteString("      Severity: " + e.Severity + "\n")
	}
	if e.Resolution != "" {
		b.WriteString("      Resolution: " + e.Resolution + "\n")
	}
	return b.String()
}

// Event is defined in Redfish schema Event.v1_4_1.json
// https://redfish.dmtf.org/schemas/v1/Event.v1_4_1.json
// The Event schema describes the JSON payload received by an Event Destination,
// which has subscribed to event notification, when events occur.  This Resource
// contains data about events, including descriptions, severity, and a MessageId
// link to a Message Registry that can be accessed for further information.
//
// Required fields: @odata.type, Events, Id, Name
type Event struct {
	// +optional
	OdataContext string `json:"@odata.context,omitempty"`
	// +required
	OdataType string `json:"@odata.type"`
	// The available actions for this resource.
	// +optional
	Actions []byte `json:"Actions,omitempty"`
	// A context can be supplied at subscription time.  This property
	// is the context value supplied by the subscriber.
	// +optional
	Context string `json:"Context,omitempty"`
	// +optional
	Description string `json:"Description,omitempty"`
	// +required
	Events []EventRecord `json:"Events"`
	// +required
	ID string `json:"Id"`
	// +required
	Name string `json:"Name"`
	// This is the manufacturer/provider specific extension
	// +optional
	Oem []byte `json:"Oem,omitempty"`
}

// String returns a pretty-printed representation of the Redfish Event.
func (e Event) String() string {
	b := strings.Builder{}
	if e.OdataContext != "" {
		b.WriteString("\n    @odata.context: " + e.OdataContext + "\n")
	}
	b.WriteString("    @odata.type: " + e.OdataType + "\n")
	if e.Actions != nil {
		b.WriteString("    Actions: " + string(e.Actions) + "\n")
	}
	if e.Context != "" {
		b.WriteString("    Context: " + e.Context + "\n")
	}
	b.WriteString("    Id: " + e.ID + "\n")
	b.WriteString("    Name: " + e.Name + "\n")
	if e.Oem != nil {
		b.WriteString("    Oem: " + string(e.Oem) + "\n")
	}
	for i, e := range e.Events {
		b.WriteString("    Events[" + strconv.Itoa(i) + "]:")
		b.WriteString(e.String())
	}
	return b.String()
}
