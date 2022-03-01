YamlTrim
========

A simple go library and utility to do better zero-trimming from complex yaml structures.

Command Usage
-------------

Install the utility:
```
go install github.com/lack/yamltrim/cmd/yamltrim
```

Yamltrim filters stdin to stdout, applying the deep trimming facility of the library to the input yaml:
```
$ cat input.yaml
top:
  middle:
    deep: ""
  other:
  - ""
  - two
$ yamltrim <input.yaml
top:
    other:
          - two
```

Library Usage
-------------

Example utility:
```
package main

import (
	"fmt"

	"github.com/lack/yamltrim"
	"gopkg.in/yaml.v3"
)

func main() {
	input := `
top:
  middle:
    deep: ""
  other:
  - ""
  - two
`

	var original interface{}
	err := yaml.Unmarshal([]byte(input), &original)
	if err != nil {
		panic(err)
	}

	trimmed := yamltrim.YamlTrim(original)

	origBytes, err := yaml.Marshal(original)
	fmt.Printf("Original:\n----------\n%s\n", origBytes)

	trimmedBytes, err := yaml.Marshal(trimmed)
	fmt.Printf("Stripped:\n----------\n%s\n", trimmedBytes)
}
```

Output:
```
Original:
----------
top:
    middle:
        deep: ""
    other:
      - ""
      - two

Stripped:
----------
top:
    other:
      - two

```
