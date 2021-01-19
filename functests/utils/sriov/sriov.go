package sriov

import (
	"os"
	"strconv"
	"time"

	g "github.com/onsi/gomega"
	sriovtestclient "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/client"
	sriovcluster "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/cluster"
)

var waitingTime time.Duration = 20 * time.Minute

func init() {
	waitingEnv := os.Getenv("SRIOV_WAITING_TIME")
	newTime, err := strconv.Atoi(waitingEnv)
	if err == nil && newTime != 0 {
		waitingTime = time.Duration(newTime) * time.Minute
	}
}

// WaitStable waits for the sriov setup to be stable after
// configuration modification.
func WaitStable(sriovclient *sriovtestclient.ClientSet) {
	// This used to be to check for sriov not to be stable first,
	// then stable. The issue is that if no configuration is applied, then
	// the status won't never go to not stable and the test will fail.
	// TODO: find a better way to handle this scenario
	time.Sleep(5 * time.Second)
	g.Eventually(func() bool {
		res, _ := sriovcluster.SriovStable("openshift-sriov-network-operator", sriovclient)
		// ignoring the error for the disconnected cluster scenario
		return res
	}, waitingTime, 1*time.Second).Should(g.BeTrue())

	g.Eventually(func() bool {
		isClusterReady, _ := sriovcluster.IsClusterStable(sriovclient)
		// ignoring the error for the disconnected cluster scenario
		return isClusterReady
	}, waitingTime, 1*time.Second).Should(g.BeTrue())
}
