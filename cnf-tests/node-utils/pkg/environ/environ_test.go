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

package environ

import (
	"fmt"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"
)

func TestDefaultFS(t *testing.T) {
	root := DefaultFS()

	if root.Sys == "" {
		t.Fatalf("missing root.sys")
	}
	if !filepath.IsAbs(root.Sys) {
		t.Fatalf("root.sys should be abspath")
	}
}

// TODO: is this more a e2e test?
func TestDefaultLog(t *testing.T) {
	root, err := getRootPath()
	if err != nil {
		t.Fatalf("cannot find the root: %v", err)
	}

	feed := "test message"
	exp := `"level"=0 "msg"="echo" "line"="test message"`

	var out strings.Builder
	cmd := exec.Command("go", "run", filepath.Join(root, "hack", "loghelper.go"))
	cmd.Stdin = strings.NewReader(feed)
	cmd.Stderr = &out // logs go on stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("run error: %v", err)
	}

	res := strings.TrimRight(out.String(), "\n\r")
	if !strings.Contains(res, exp) {
		t.Fatalf("unexpected output: got=<<%s>> exp=<<%s>>", res, exp)
	}
}

func TestNew(t *testing.T) {
	env := New()

	// TODO: what about the rest?
	if !filepath.IsAbs(env.Root.Sys) {
		t.Fatalf("root.sys should be abspath")
	}
}

func getRootPath() (string, error) {
	_, file, _, ok := goruntime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot retrieve tests directory")
	}
	basedir := filepath.Dir(file)
	return filepath.Abs(filepath.Join(basedir, "..", ".."))
}
