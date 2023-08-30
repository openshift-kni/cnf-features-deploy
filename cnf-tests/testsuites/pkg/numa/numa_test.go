package numa

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var lsCpu2NUMA16CoresOutput string = strings.ReplaceAll(`# The following is the parsable format, which can be fed to other
# programs. Each different item in every column has an unique ID
# starting from zero.
# CPU,Node
0,0
1,1
2,0
3,1
4,0
5,1
6,0
7,1
8,0
9,1
10,0
11,1
12,0
13,1
14,0
15,1
`, "\n", "\r\n")

func TestFindNUMAForCPUsInLscpu(t *testing.T) {
	result, err := findForCPUsParseOutput(lsCpu2NUMA16CoresOutput, []string{"0", "2", "4", "6"})
	assert.NoError(t, err)
	assert.Equal(t, 0, result)

	result, err = findForCPUsParseOutput(lsCpu2NUMA16CoresOutput, []string{"1", "3"})
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	_, err = findForCPUsParseOutput(lsCpu2NUMA16CoresOutput, []string{"0", "1"})
	assert.Error(t, err)
}
