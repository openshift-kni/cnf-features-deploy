package l2client

import (
	daemonsets "github.com/test-network-function/privileged-daemonset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type L2Client struct {
	K8sClient kubernetes.Interface
	Rest      *rest.Config
}

var Client = L2Client{}

func Set(k8sClient kubernetes.Interface, restClient *rest.Config) {
	Client.K8sClient = k8sClient
	Client.Rest = restClient
	daemonsets.SetDaemonSetClient(Client.K8sClient)
}
