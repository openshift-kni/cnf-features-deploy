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
 * Copyright 2021 Red Hat, Inc.
 */

// flagcodec allows to manipulate foreign command lines following the
// standard golang conventions. It offeres two-way processing aka
// parsing/marshalling of flags. It is different from the other, well
// established packages (pflag...) because it aims to manipulate command
// lines in general, not this program command line.

package flagcodec

import (
	"fmt"
	"strings"
)

const (
	FlagToggle = iota
	FlagOption
)

type Val struct {
	Kind int
	Data string
}

type Flags struct {
	command string
	args    map[string]Val
	keys    []string
}

func ParseArgvKeyValue(args []string) *Flags {
	return ParseArgvKeyValueWithCommand("", args)
}

// ParseArgvKeyValue parses a clean (trimmed) argv whose components are
// either toggles or key=value pairs. IOW, this is a restricted and easier
// to parse flavour of argv on which option and value are guaranteed to
// be in the same item.
// IOW, we expect
// "--opt=foo"
// AND NOT
// "--opt", "foo"
func ParseArgvKeyValueWithCommand(command string, args []string) *Flags {
	ret := &Flags{
		command: command,
		args:    make(map[string]Val),
	}
	for _, arg := range args {
		fields := strings.SplitN(arg, "=", 2)
		if len(fields) == 1 {
			ret.SetToggle(fields[0])
			continue
		}
		ret.SetOption(fields[0], fields[1])
	}
	return ret
}

func (fl *Flags) recordFlag(name string) {
	if _, ok := fl.args[name]; !ok {
		fl.keys = append(fl.keys, name)
	}
}

func (fl *Flags) SetToggle(name string) {
	fl.recordFlag(name)
	fl.args[name] = Val{
		Kind: FlagToggle,
	}
}

func (fl *Flags) SetOption(name, data string) {
	fl.recordFlag(name)
	fl.args[name] = Val{
		Kind: FlagOption,
		Data: data,
	}
}

func (fl *Flags) Command() string {
	return fl.command
}

func (fl *Flags) Args() []string {
	var args []string
	for _, name := range fl.keys {
		args = append(args, toString(name, fl.args[name]))
	}
	return args
}

func (fl *Flags) Argv() []string {
	args := fl.Args()
	if fl.command == "" {
		return args
	}
	return append([]string{fl.Command()}, args...)
}

func (fl *Flags) GetFlag(name string) (Val, bool) {
	if val, ok := fl.args[name]; ok {
		return val, ok
	}
	return Val{}, false
}

func toString(name string, val Val) string {
	if val.Kind == FlagToggle {
		return name
	}
	return fmt.Sprintf("%s=%s", name, val.Data)
}
