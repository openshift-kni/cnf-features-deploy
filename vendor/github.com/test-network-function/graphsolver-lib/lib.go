package lib

import (
	"github.com/sirupsen/logrus"
	export "github.com/test-network-function/graphsolver-exports"
	l2lib "github.com/test-network-function/l2discovery-exports"
)

var GlobalConfig = configObject{}

type configObject struct {
	// tag for problem variables
	variableTagToInt map[string][]int
	// problem definition
	problems map[string][][][]int
	// map storing solutions
	solutions map[string]*[][]int
	l2Config  export.L2Info
}

func (config *configObject) SetL2Config(param export.L2Info) {
	config.l2Config = param
}
func (config *configObject) GetSolutions() map[string]*[][]int {
	return config.solutions
}
func (config *configObject) InitProblem(name string, problemStatement [][][]int, problemVariablesMapping []int) {
	if GlobalConfig.problems == nil {
		GlobalConfig.problems = make(map[string][][][]int)
	}
	GlobalConfig.problems[name] = problemStatement
	if GlobalConfig.variableTagToInt == nil {
		GlobalConfig.variableTagToInt = make(map[string][]int)
	}
	GlobalConfig.variableTagToInt[name] = problemVariablesMapping
	if GlobalConfig.solutions == nil {
		GlobalConfig.solutions = make(map[string]*[][]int)
	}
	GlobalConfig.solutions[name] = &[][]int{}
}

func (config *configObject) Run(problemName string) {
	// get all the vertices in the graph
	L := GetAllGraphVertices(len(config.l2Config.GetPtpIfList()))

	// Running solver
	PermutationsWithConstraints(config.l2Config, GlobalConfig.problems[problemName], L, 0, len(GlobalConfig.problems[problemName]), len(L), true, GlobalConfig.solutions[problemName])
}

// Prints the all solutions for each scenario, if found
func (config configObject) PrintSolutions(all bool) {
	for index, solutions := range config.solutions {
		if len(*solutions) == 0 {
			logrus.Infof("Solution for %s problem does not exists", index)
			continue
		}
		logrus.Infof("Solutions for %s problem", index)
		for _, solution := range *solutions {
			PrintSolution(config.l2Config, solution)
			logrus.Infof("---")
			if !all {
				break
			}
		}
	}
}

// Prints the first solution for each scenario, if found
func (config configObject) PrintFirstSolution() {
	config.PrintSolutions(false)
}

// Prints the all solutions for each scenario, if found
func (config configObject) PrintAllSolutions() {
	config.PrintSolutions(true)
}

// list of Algorithm functions with zero params
type AlgoFunction0 int

// See applyStep
const (
	// same node
	StepNil AlgoFunction0 = iota
)

// list of Algorithm function with 1 params
type AlgoFunction1 int

// See applyStep
const (
	// same node
	StepIsPTP AlgoFunction1 = iota
)

// list of Algorithm function with 2 params
type AlgoFunction2 int

// See applyStep
const (
	StepSameLan2 AlgoFunction2 = iota
	StepSameNic
	StepSameNode
	StepDifferentNode
	StepDifferentNic
)

// list of Algorithm function with 3 params
type AlgoFunction3 int

// See applyStep
const (
	StepSameLan3 AlgoFunction3 = iota
)

// Signature for algorithm functions with 0 params
type ConfigFunc0 func() bool

// Signature for algorithm functions with 1 params
type ConfigFunc1 func(export.L2Info, int) bool

// Signature for algorithm functions with 2 params
type ConfigFunc2 func(export.L2Info, int, int) bool

// Signature for algorithm functions with 3 params
type ConfigFunc3 func(export.L2Info, int, int, int) bool

type ConfigFunc func(export.L2Info, []int) bool
type Algorithm struct {
	// number of interfaces to solve
	IfCount int
	// Function to run algo
	TestSolution ConfigFunc
}

// Print a single solution
func PrintSolution(config export.L2Info, p []int) {
	i := 0
	for _, aIf := range p {
		logrus.Infof("p%d= %s", i, config.GetPtpIfList()[aIf])
		i++
	}
}

func GetAllGraphVertices(count int) (l []int) {
	for i := 0; i < count; i++ {
		l = append(l, i)
	}
	return l
}

// Recursive solver function. Creates a set of permutations and applies contraints at each step to
// reduce the solution graph and speed up execution
func PermutationsWithConstraints(config export.L2Info, algo [][][]int, l []int, s, e, n int, result bool, solutions *[][]int) {
	if !result || len(l) < e {
		return
	}
	if s == e {
		temp := make([]int, 0)
		temp = append(temp, l...)
		temp = temp[0:e]
		logrus.Debugf("%v --  %v", temp, result)
		*solutions = append(*solutions, temp)
	} else {
		// Backtracking loop
		for i := s; i < n; i++ {
			l[i], l[s] = l[s], l[i]
			result = applyStep(config, algo[s], l[0:e])
			PermutationsWithConstraints(config, algo, l, s+1, e, n, result, solutions)
			l[i], l[s] = l[s], l[i]
		}
	}
}

// check if an interface is receiving GM
func IsPTP(config export.L2Info, aInterface *l2lib.PtpIf) bool {
	for _, aIf := range config.GetPortsGettingPTP() {
		if aInterface.IfClusterIndex == aIf.IfClusterIndex {
			return true
		}
	}
	return false
}

// Checks that an if an interface receives ptp frames
func IsPTPWrapper(config export.L2Info, if1 int) bool {
	return IsPTP(config, config.GetPtpIfList()[if1])
}

// Checks if 2 interfaces are on the same node
func SameNode(if1, if2 *l2lib.PtpIf) bool {
	return if1.NodeName == if2.NodeName
}

// algo Wrapper for SameNode
func SameNodeWrapper(config export.L2Info, if1, if2 int) bool {
	return SameNode(config.GetPtpIfList()[if1], config.GetPtpIfList()[if2])
}

// algo wrapper for !SameNode
func DifferentNodeWrapper(config export.L2Info, if1, if2 int) bool {
	return !SameNode(config.GetPtpIfList()[if1], config.GetPtpIfList()[if2])
}

// Algo wrapper for !SameNic
func DifferentNicWrapper(config export.L2Info, if1, if2 int) bool {
	return !SameNic(config.GetPtpIfList()[if1], config.GetPtpIfList()[if2])
}

// Checks if 3 interfaces are connected to the same LAN
func SameLan3(config export.L2Info, if1, if2, if3 int, lans *[][]int) bool {
	if SameNode(config.GetPtpIfList()[if1], config.GetPtpIfList()[if2]) ||
		SameNode(config.GetPtpIfList()[if1], config.GetPtpIfList()[if3]) {
		return false
	}
	for _, Lan := range *lans {
		if1Present := false
		if2Present := false
		if3Present := false
		for _, aIf := range Lan {
			if aIf == if1 {
				if1Present = true
			}
			if aIf == if2 {
				if2Present = true
			}
			if aIf == if3 {
				if3Present = true
			}
		}
		if if1Present && if2Present && if3Present {
			return true
		}
	}
	return false
}

// algo wrapper for SameLan3
func SameLan3Wrapper(config export.L2Info, if1, if2, if3 int) bool {
	return SameLan3(config, if1, if2, if3, config.GetLANs())
}

// Checks if 2 interfaces are connected to the same LAN
func SameLan2(config export.L2Info, if1, if2 int, lans *[][]int) bool {
	if SameNode(config.GetPtpIfList()[if1], config.GetPtpIfList()[if2]) {
		return false
	}
	for _, Lan := range *lans {
		if1Present := false
		if2Present := false
		for _, aIf := range Lan {
			if aIf == if1 {
				if1Present = true
			}
			if aIf == if2 {
				if2Present = true
			}
		}
		if if1Present && if2Present {
			return true
		}
	}
	return false
}

// wrapper for SameLan2
func SameLan2Wrapper(config export.L2Info, if1, if2 int) bool {
	return SameLan2(config, if1, if2, config.GetLANs())
}

// Determines if 2 interfaces (ports) belong to the same NIC
func SameNic(ifaceName1, ifaceName2 *l2lib.PtpIf) bool {
	if ifaceName1.IfClusterIndex.NodeName != ifaceName2.IfClusterIndex.NodeName {
		return false
	}
	return ifaceName1.IfPci.Device != "" && ifaceName1.IfPci.Device == ifaceName2.IfPci.Device
}

// wrapper for SameNic
func SameNicWrapper(config export.L2Info, if1, if2 int) bool {
	return SameNic(config.GetPtpIfList()[if1], config.GetPtpIfList()[if2])
}

// wrapper for nil algo function
func NilWrapper() bool {
	return true
}

// Applies a single step (constraint) in the backtracking algorithm
func applyStep(config export.L2Info, step [][]int, combinations []int) bool {
	type paramNum int

	const (
		NoParam paramNum = iota
		OneParam
		TwoParams
		ThreeParams
		FourParams
	)
	// mapping table between :
	// AlgoFunction0, AlgoFunction1, AlgoFunction2, AlgoFunction3 and
	// function wrappers

	var AlgoCode0 [1]ConfigFunc0
	AlgoCode0[StepNil] = NilWrapper

	var AlgoCode1 [1]ConfigFunc1
	AlgoCode1[StepIsPTP] = IsPTPWrapper

	var AlgoCode2 [5]ConfigFunc2
	AlgoCode2[StepSameLan2] = SameLan2Wrapper
	AlgoCode2[StepSameNic] = SameNicWrapper
	AlgoCode2[StepSameNode] = SameNodeWrapper
	AlgoCode2[StepDifferentNode] = DifferentNodeWrapper
	AlgoCode2[StepDifferentNic] = DifferentNicWrapper

	var AlgoCode3 [1]ConfigFunc3
	AlgoCode3[StepSameLan3] = SameLan3Wrapper

	result := true
	for _, test := range step {
		switch test[1] {
		case int(NoParam):
			result = result && AlgoCode0[test[0]]()
		case int(OneParam):
			result = result && AlgoCode1[test[0]](config, combinations[test[2]])
		case int(TwoParams):
			result = result && AlgoCode2[test[0]](config, combinations[test[2]], combinations[test[3]])
		case int(ThreeParams):
			result = result && AlgoCode3[test[0]](config, combinations[test[2]], combinations[test[3]], combinations[test[4]])
		}
	}
	return result
}
