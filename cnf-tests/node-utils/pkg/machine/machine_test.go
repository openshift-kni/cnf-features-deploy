/*
 * Copyright 2024 Red Hat, Inc.
 *
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
 */

package machine

import (
	"fmt"
	"path/filepath"
	goruntime "runtime"
	"testing"

	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/node-utils/pkg/environ"
)

func TestDiscoverFromFile(t *testing.T) {
	cur, err := getCurrentPath()
	if err != nil {
		t.Fatalf("failed to get current path: %v", err)
	}
	env := environ.New()
	env.DataPath = filepath.Join(cur, "testdata", "machine_laptop.json")
	got, err := Discover(env)
	if err != nil {
		t.Fatalf("discover error against real machine: %v", err)
	}
	if got.CPU == nil || got.Topology == nil {
		t.Fatalf("missing expected data in machine info CPU=%v Topology=%v", got.CPU, got.Topology)
	}
}

func TestDiscoverFundamentals(t *testing.T) {
	env := environ.New()
	got, err := Discover(env)
	if err != nil {
		t.Fatalf("discover error against real machine: %v", err)
	}
	if got.CPU == nil || got.Topology == nil {
		t.Fatalf("missing expected data in machine info CPU=%v Topology=%v", got.CPU, got.Topology)
	}
}

func TestMachineJSONRoundtrip(t *testing.T) {
	env := environ.New()
	got, err := Discover(env)
	if err != nil {
		t.Fatalf("discover error against real machine: %v", err)
	}

	refJSON, err := got.ToJSON()
	if err != nil {
		t.Fatalf("failed to convert ref to JSON: %v", err)
	}
	aux, err := FromJSON(refJSON)
	if err != nil {
		t.Fatalf("failed to recover from JSON: %v", err)
	}
	auxJSON, err := aux.ToJSON()
	if err != nil {
		t.Fatalf("failed to convert aux to JSON: %v", err)
	}

	if refJSON != auxJSON {
		t.Errorf("JSON mismatch, Machines are not equal.\nref=%s\naux=%s", refJSON, auxJSON)
	}
}

func getCurrentPath() (string, error) {
	_, file, _, ok := goruntime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot retrieve tests directory")
	}
	return filepath.Dir(file), nil
}
