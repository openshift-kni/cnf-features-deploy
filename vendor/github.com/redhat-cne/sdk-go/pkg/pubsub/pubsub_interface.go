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

package pubsub

// Reader is the interface for reading through an event from attributes.
type Reader interface {
	// GetResource returns event.GetResource()
	GetResource() string
	// GetEndpointUri returns event.GetEndpointUri()
	GetEndpointURI() string
	// GetURILocation returns event.GetUriLocation()
	GetURILocation() string
	GetID() string
	// String returns a pretty-printed representation of the PubSub.
	String() string
}

// Writer is the interface for writing through an event onto attributes.
// If an error is thrown by a sub-component, Writer caches the error
// internally and exposes errors with a call to Writer.Validate().
type Writer interface {
	// SetResource performs event.SetResource()
	SetResource(string) error
	// SetEndpointURI [erforms] event.SetEndpointURI()
	SetEndpointURI(string) error
	// SetURILocation performs event.SetURILocation()
	SetURILocation(string) error
	// SetID performs event.SetID.
	SetID(string)
}
