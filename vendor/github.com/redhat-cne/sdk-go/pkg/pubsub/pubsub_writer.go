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

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/redhat-cne/sdk-go/pkg/types"
)

var _ Writer = (*PubSub)(nil)

// SetResource implements EventWriter.SetResource
func (ps *PubSub) SetResource(s string) error {
	matched, err := regexp.MatchString(`([^/]+(/{2,}[^/]+)?)`, s)
	if matched {
		ps.Resource = s
	} else {
		return err
	}
	return nil
}

// SetID implements EventWriter.SetID
func (ps *PubSub) SetID(id string) {
	ps.ID = id
}

// SetEndpointURI ...
func (ps *PubSub) SetEndpointURI(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		ps.EndPointURI = nil
		err := fmt.Errorf("uriLocation is given empty string,should be valid url")
		return err
	}
	pu, err := url.Parse(s)
	if err != nil {
		return err
	}
	ps.EndPointURI = &types.URI{URL: *pu}
	return nil
}

// SetURILocation  sets uri location attribute
func (ps *PubSub) SetURILocation(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		ps.URILocation = nil
		err := fmt.Errorf("uriLocation is given empty string,should be valid url")
		return err
	}
	pu, err := url.Parse(s)
	if err != nil {
		return err
	}
	ps.URILocation = &types.URI{URL: *pu}
	return nil
}
