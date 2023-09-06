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

const (
	// TextPlain ...
	TextPlain = "text/plain"
	// TextJSON ...
	TextJSON = "text/json"
	// ApplicationJSON ...
	ApplicationJSON = "application/json"
)

// StringOfApplicationJSON returns a string pointer to "application/json"
func StringOfApplicationJSON() *string {
	a := ApplicationJSON
	return &a
}

// StringOfTextPlain returns a string pointer to "text/plain"
func StringOfTextPlain() *string {
	a := TextPlain
	return &a
}
