package libexports

import (
	l2lib "github.com/test-network-function/l2discovery-exports"
)

type SolverConfig interface {
	// problem definition
	InitProblems(string, [][][]int, []int)
	// L2 configuration
	SetL2Config(L2Info)
	// Run solver on problem
	Run(string)
	// Prints all solutions
	PrintAllSolutions()
	// map storing solutions
	GetSolutions() map[string]*[][]int
}

type L2Info interface {
	// list of cluster interfaces indexed with a simple integer (X) for readability in the graph
	GetPtpIfList() []*l2lib.PtpIf
	// LANs identified in the graph
	GetLANs() *[][]int
	// List of port receiving PTP frames (assuming valid GM signal received)
	GetPortsGettingPTP() []*l2lib.PtpIf
}
