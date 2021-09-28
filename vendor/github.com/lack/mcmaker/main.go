package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	mcmaker "github.com/lack/mcmaker/pkg"
)

type command struct {
	name       string
	run        func([]string, mcmaker.McMaker) ([]string, error)
	shortusage string
}

func addFile(args []string, m mcmaker.McMaker) ([]string, error) {
	c := flag.NewFlagSet("file", flag.ExitOnError)
	c.Usage = func() {
		o := flag.CommandLine.Output()
		fmt.Fprintf(o, "Adds a file to the MachineConfig object\n\nUsage:\n  %s ... file [options] ...\n\nOptions:\n", os.Args[0])
		c.PrintDefaults()
	}
	source := c.String("source", "", "The local file containing the file data")
	path := c.String("path", "", "Path and filename to create on the destination host")
	mode := c.Int("mode", 0644, "mode to create")
	err := c.Parse(args)
	if err != nil {
		return nil, err
	}
	err = m.AddFile(*source, *path, *mode)
	if err != nil {
		return nil, err
	}
	return c.Args(), nil
}

func addUnit(args []string, m mcmaker.McMaker) ([]string, error) {
	c := flag.NewFlagSet("unit", flag.ExitOnError)
	c.Usage = func() {
		o := flag.CommandLine.Output()
		fmt.Fprintf(o, "Adds a systemd unit to the MachineConfig object\n\nUsage:\n  %s ... unit [options] ...\n\nOptions:\n", os.Args[0])
		c.PrintDefaults()
	}
	source := c.String("source", "", "The local file containing the unit definition")
	name := c.String("name", "", "Unit name to create (defaults to basename of source)")
	enable := c.Bool("enable", true, "Should it be enabled")
	err := c.Parse(args)
	if err != nil {
		return nil, err
	}
	err = m.AddUnit(*source, *name, *enable)
	if err != nil {
		return nil, err
	}
	return c.Args(), nil
}

func addDropin(args []string, m mcmaker.McMaker) ([]string, error) {
	c := flag.NewFlagSet("dropin", flag.ExitOnError)
	c.Usage = func() {
		o := flag.CommandLine.Output()
		fmt.Fprintf(o, "Adds a systemd drop-in to the MachineConfig object\n\nUsage:\n  %s ... dropin [options] ...\n\nOptions:\n", os.Args[0])
		c.PrintDefaults()
	}
	source := c.String("source", "", "The local file containing the drop-in definition")
	servicename := c.String("for", "", "The name of the service to attach to the drop-in")
	name := c.String("name", "", "The drop-in name to create (defaults to basename of source)")
	err := c.Parse(args)
	if err != nil {
		return nil, err
	}
	err = m.AddDropin(*source, *servicename, *name)
	if err != nil {
		return nil, err
	}
	return c.Args(), nil
}

func main() {
	commands := map[string]command{
		"file": {
			name:       "file",
			run:        addFile,
			shortusage: "file -source file -path /path [-mode 0644]",
		},
		"unit": {
			name:       "unit",
			run:        addUnit,
			shortusage: "unit -source file [-name name] [-enable=false]",
		},
		"dropin": {
			name:       "dropin",
			run:        addDropin,
			shortusage: "dropin -source file -for servicename [-name name]",
		},
	}

	flag.Usage = func() {
		o := flag.CommandLine.Output()
		fmt.Fprintf(o, "Creates a MachineConfig object with custom contents\n\nUsage:\n  %s [options] [commands...]\n\nOptions:\n", os.Args[0])
		flag.CommandLine.PrintDefaults()
		fmt.Fprintf(o, "\nCommands:\n")
		for _, cmd := range commands {
			fmt.Fprintf(o, "  %s\n", cmd.shortusage)
		}
		fmt.Fprintf(o, "\nRun '%s help command' for details on each specific command\n", os.Args[0])
	}
	name := flag.String("name", "", "The name of the MC object to create")
	stdout := flag.Bool("stdout", false, "If set, dump the object to stdout.  If not, creates a file called 'name.yaml' based on '-name'")
	role := flag.String("mcp", "master,worker", "The MCP role(s) to select (comma-delimited)")
	flag.Parse()

	if flag.Arg(0) == "help" {
		cmd := flag.Arg(1)
		handler, ok := commands[cmd]
		if ok {
			handler.run([]string{"-help"}, mcmaker.McMaker{})
		} else {
			flag.Usage()
			os.Exit(0)
		}
	}

	if *name == "" {
		fmt.Fprintf(flag.CommandLine.Output(), "No -name was specified\n\n")
		flag.Usage()
		os.Exit(1)
	}
	m := mcmaker.New(*name)

	remaining := flag.Args()
	var err error
	for len(remaining) > 0 {
		handler, ok := commands[remaining[0]]
		if ok {
			remaining, err = handler.run(remaining[1:], m)
			if err != nil {
				panic(err)
			}
		} else {
			fmt.Fprintf(flag.CommandLine.Output(), "Unrecognized command %q\n\n", remaining[0])
			flag.Usage()
			os.Exit(1)
		}
	}

	var output io.Writer
	if *stdout {
		output = os.Stdout
	} else {
		output, err = os.Create(fmt.Sprintf("%s.yaml", *name))
		if err != nil {
			panic(err)
		}
	}
	roles := strings.Split(*role, ",")
	for _, r := range roles {
		if len(roles) > 1 {
			output.Write([]byte("---\n"))
		}
		m.SetRole(r)
		m.WriteTo(output)
	}
}
