# mcmaker
McMaker makes [MachineConfig](https://github.com/openshift/machine-config-operator) objects

## Installation

```bash
go install github.com/lack/mcmaker@latest
```

## Usage

```
Creates a MachineConfig object with custom contents

Usage:
  ./mcmaker [options] [commands...]

Options:
  -mcp string
    	The MCP role(s) to select (comman-delimited) (default "master,worker")
  -name string
    	The name of the MC object to create
  -stdout
    	If set, dump the object to stdout.  If not, creates a file called 'name.yaml' based on '-name'

Commands:
  file -source file -path /path [-mode 0644]
  unit -source file [-name name] [-enable=false]

Run './mcmaker help command' for details on each specific command
```

For example, to construct a MachineConfig object with one executable and two systemd units, you could run something like this:

```bash
mcmaker -name 42-custom-service \
  file -source executable.sh -path /usr/local/bin/executable.sh -mode 0755 \
  unit -source one.service \
  unit -source other.service
```

By default this will build a pair of MachineConfig objects, one each for 'master' and 'worker' roles, and output them to a file called `42-custom-service.yaml`

