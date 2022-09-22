/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2022 Red Hat, Inc.
 */

package platform

import goversion "github.com/aquasecurity/go-version/pkg/version"

type Version string

const MissingVersion Version = ""

func ParseVersion(v string) (Version, error) {
	_, err := goversion.Parse(v)
	if err != nil {
		return Version(""), err
	}
	return Version(v), nil
}

func (v Version) String() string {
	return string(v)
}

func (v Version) AtLeastString(other string) (bool, error) {
	ref, err := goversion.Parse(other)
	if err != nil {
		return false, err
	}
	ser, err := goversion.Parse(v.String())
	if err != nil {
		return false, err
	}
	return ser.Compare(ref) >= 0, nil
}

func (v Version) AtLeast(other Version) (bool, error) {
	return v.AtLeastString(other.String())
}
