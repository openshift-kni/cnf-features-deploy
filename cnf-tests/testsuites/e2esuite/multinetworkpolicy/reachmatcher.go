package multinetworkpolicy

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"

	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
	client "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/images"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/pods"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const ipTablesDebugContainerName = "debug-iptable-container"

// AddTCPNetcatServerToPod adds a container to the pod with a TCP netcat server that are suitable to be used with ReachMatcher.
func AddTCPNetcatServerToPod(pod *corev1.Pod, port intstr.IntOrString) *corev1.Pod {
	// --keep-open parameters helps to use the same server for multiple tests. Incoming messages
	// are forwarder to pod logs (stdout/stderr) and they are used to validate connectivity (see `canSendTraffic``)
	pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{
		Name:    "netcat-tcp-server-" + port.String(),
		Image:   images.For(images.TestUtils),
		Command: []string{"nc", "-vv", "--keep-open", "--listen", port.String()},
	})

	return pod
}

// AddUDPNetcatServerToPod adds a container to the pod with a UDP netcat server that are suitable to be used with ReachMatcher.
func AddUDPNetcatServerToPod(pod *corev1.Pod, port intstr.IntOrString) *corev1.Pod {
	// UDP servers support --keep-open only with --sh-exec option, and to get the incoming messages
	// to logs they are needed to be sent to stderr, as stdout is redirected back to the client
	pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{
		Name:    "netcat-udp-server-" + port.String(),
		Image:   images.For(images.TestUtils),
		Command: []string{"nc", "-vv", "--udp", "--keep-open", "--sh-exec", "/bin/cat >&2", "--listen", port.String()},
	})
	return pod
}

// AddSCTPNetcatServerToPod adds a container to the pod with a SCTP netcat server that are suitable to be used with ReachMatcher.
func AddSCTPNetcatServerToPod(pod *corev1.Pod, port intstr.IntOrString) *corev1.Pod {
	pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{
		Name:    "netcat-sctp-server-" + port.String(),
		Image:   images.For(images.TestUtils),
		Command: []string{"nc", "-vv", "--sctp", "--keep-open", "--sh-exec", "/bin/cat >&2", "--listen", port.String()},
	})
	return pod
}

// AddIPTableDebugContainer adds a container that polls iptables information and print them to stdout
func AddIPTableDebugContainer(pod *corev1.Pod) *corev1.Pod {
	pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{
		Name:            ipTablesDebugContainerName,
		Image:           images.For(images.TestUtils),
		Command:         []string{"sh", "-c", "while true; do iptables -L -v -n; sleep 10; done"},
		SecurityContext: &corev1.SecurityContext{Privileged: ptr.To(true)}})

	return pod
}

// ReachMatcher allows making assertion on pod connectivity using netcat servers and clients
type ReachMatcher struct {
	destinationPod     *corev1.Pod
	destinationPort    string
	destinationAddress string
	protocol           corev1.Protocol
	ipFamily           corev1.IPFamily
}

// ReachOpt describe a function that applies to a ReachMatcher
type ReachOpt func(*ReachMatcher)

// Reach creates a new ReachMatcher with a destination pod and a list of options.
// Default options include using TCP traffic, 5555 as destination port and IPv4 as ip family.
func Reach(destinationPod *corev1.Pod, opts ...ReachOpt) types.GomegaMatcher {
	ret := &ReachMatcher{
		destinationPod:  destinationPod,
		destinationPort: port5555.String(),
		protocol:        corev1.ProtocolTCP,
		ipFamily:        corev1.IPv4Protocol,
	}

	for _, opt := range opts {
		opt(ret)
	}

	ret.destinationAddress = getMultusNicIP(ret.destinationPod, ret.ipFamily)

	return ret
}

// OnPort specifies the destination port to be used in the matcher.
func OnPort(port intstr.IntOrString) ReachOpt {
	return func(s *ReachMatcher) {
		s.destinationPort = port.String()
	}
}

// ViaTCP specifies that the TCP protocol must be used to check the connectivity.
var ViaTCP ReachOpt = func(s *ReachMatcher) {
	s.protocol = corev1.ProtocolTCP
}

// ViaUDP specifies that the UDP protocol must be used to check the connectivity.
var ViaUDP ReachOpt = func(s *ReachMatcher) {
	s.protocol = corev1.ProtocolUDP
}

// ViaSCTP specifies that the SCTP protocol must be used to check the connectivity.
var ViaSCTP ReachOpt = func(s *ReachMatcher) {
	s.protocol = corev1.ProtocolSCTP
}

// ViaIPv4 specifies to use v4 as IP family.
var ViaIPv4 ReachOpt = func(s *ReachMatcher) {
	s.ipFamily = corev1.IPv4Protocol
}

// ViaIPv6 specifies to use v6 as IP family.
var ViaIPv6 ReachOpt = func(s *ReachMatcher) {
	s.ipFamily = corev1.IPv6Protocol
}

// Match checks if the actual meets the Reach condition.
func (m *ReachMatcher) Match(actual interface{}) (bool, error) {
	sourcePod, ok := actual.(*corev1.Pod)
	if !ok {
		return false, fmt.Errorf("ReachMatcher must be passed an *Pod. Got\n%s", format.Object(actual, 1))
	}

	return canSendTraffic(sourcePod, m.destinationPod, m.destinationPort, m.ipFamily, m.protocol)
}

func (m *ReachMatcher) FailureMessage(actual interface{}) string {
	sourcePod, ok := actual.(*corev1.Pod)
	if !ok {
		return "ReachMatcher should be used against v1.Pod objects"
	}

	return fmt.Sprintf(`pod [%s/%s %s] is not reachable by pod [%s/%s] on port[%s:%s], but it should be.
%s
Server iptables:
%s
-----
Client iptables:
%s`,
		m.destinationPod.Namespace, m.destinationPod.Name, m.destinationAddress,
		sourcePod.Namespace, sourcePod.Name,
		m.protocol, m.destinationPort,
		makeConnectivityMatrix(m.destinationPort, m.ipFamily, m.protocol,
			nsX_podA, nsX_podB, nsX_podC,
			nsY_podA, nsY_podB, nsY_podC,
			nsZ_podA, nsZ_podB, nsZ_podC,
		),
		getIPTables(m.destinationPod, m.ipFamily),
		getIPTables(sourcePod, m.ipFamily),
	)
}

// NegatedFailureMessage builds the message to show in case of negated assertion failure
func (m *ReachMatcher) NegatedFailureMessage(actual interface{}) string {
	sourcePod, ok := actual.(*corev1.Pod)
	if !ok {
		return "ReachMatcher should be used against v1.Pod objects"
	}

	return fmt.Sprintf(`pod [%s/%s %s] is reachable by pod [%s/%s] on port[%s:%s], but it shouldn't be.
%s
Server iptables:
%s
-----
Client iptables:
%s`,
		m.destinationPod.Namespace, m.destinationPod.Name, m.destinationAddress,
		sourcePod.Namespace, sourcePod.Name,
		m.protocol, m.destinationPort,
		makeConnectivityMatrix(m.destinationPort, m.ipFamily, m.protocol,
			nsX_podA, nsX_podB, nsX_podC,
			nsY_podA, nsY_podB, nsY_podC,
			nsZ_podA, nsZ_podB, nsZ_podC,
		),
		getIPTables(m.destinationPod, m.ipFamily),
		getIPTables(sourcePod, m.ipFamily),
	)
}

func canSendTraffic(sourcePod, destinationPod *corev1.Pod, destinationPort string, ipFamily corev1.IPFamily, protocol corev1.Protocol) (bool, error) {
	destinationIP := getMultusNicIP(destinationPod, ipFamily)

	protocolArg := ""
	if protocol == corev1.ProtocolUDP {
		protocolArg = "--udp"
	}

	if protocol == corev1.ProtocolSCTP {
		protocolArg = "--sctp"
	}

	saltString := fmt.Sprintf("%d", rand.Intn(1000000)+1000000)

	containerName, err := findContainerNameByImage(sourcePod, images.For(images.TestUtils))
	if err != nil {
		return false, fmt.Errorf("can't check connectivity from source pod [%s]: %w", sourcePod.Name, err)
	}

	output, err := pods.ExecCommandInContainer(
		client.Client,
		*sourcePod,
		containerName,
		[]string{
			"bash", "-c",
			fmt.Sprintf("echo '%s (%s/%s)-%s:%s%s' | nc -w 1 %s %s %s",
				saltString,
				sourcePod.Namespace, sourcePod.Name,
				destinationIP,
				destinationPort,
				protocol,
				protocolArg,
				destinationIP,
				destinationPort,
			),
		})

	if err != nil {
		if doesErrorMeanNoConnectivity(output.String(), protocol) {
			return false, nil
		}

		return false, fmt.Errorf("can't connect pods [%s] -> [%s]: %w\nServer iptables\n%s\n---\nClient iptables\n%s",
			sourcePod.Name, destinationPod.Name, err, getIPTables(destinationPod, ipFamily), getIPTables(sourcePod, ipFamily))
	}

	destinationContainerName := fmt.Sprintf("netcat-%s-server-%s", strings.ToLower(string(protocol)), destinationPort)
	destinationLogs, err := pods.GetLogForContainer(
		destinationPod,
		destinationContainerName,
	)
	if err != nil {
		return false, fmt.Errorf("can't get destination pod logs [%s/%s]: %w ", destinationPod.Name, destinationContainerName, err)
	}

	if strings.Contains(destinationLogs, saltString) {
		return true, nil
	}
	return false, nil
}

func doesErrorMeanNoConnectivity(commandOutput string, protocol corev1.Protocol) bool {
	// Since v7.92, ncat timeout error message has changed.
	// See https://github.com/nmap/nmap/commit/4824a5a0742a77f92c43eb5c9d9c420d56dbadcc
	const NCAT_v7_70_TIMEOUT string = "Ncat: Connection timed out"
	const NCAT_v7_92_TIMEOUT string = "Ncat: TIMEOUT"

	switch protocol {
	case corev1.ProtocolTCP:
		if strings.Contains(commandOutput, NCAT_v7_70_TIMEOUT) ||
			strings.Contains(commandOutput, NCAT_v7_92_TIMEOUT) {
			// Timeout error is symptom of no connection
			return true
		}
	case corev1.ProtocolSCTP:
		if strings.Contains(commandOutput, NCAT_v7_70_TIMEOUT) ||
			strings.Contains(commandOutput, NCAT_v7_92_TIMEOUT) {
			// Timeout error is symptom of no connection
			return true
		}
	case corev1.ProtocolUDP:
		if strings.Contains(commandOutput, "Ncat: Connection refused") {
			return true
		}
	}

	return false
}

func getIPTables(pod *corev1.Pod, ipFamily corev1.IPFamily) string {
	iptablesCmd := "iptables"
	if ipFamily == corev1.IPv6Protocol {
		iptablesCmd = "ip6tables"
	}

	output, err := pods.ExecCommandInContainer(client.Client, *pod, ipTablesDebugContainerName, []string{iptablesCmd, "-L", "-v", "-n"})
	if err != nil {
		return "<err: " + err.Error() + ">"
	}

	return output.String()
}

func getMultusNicIP(pod *corev1.Pod, ipFamily corev1.IPFamily) string {
	ips, err := getNicIPs(pod, "net1")
	if err != nil {
		return "<err: " + err.Error() + ">"
	}

	if len(ips) == 0 {
		return "<no IPs>"
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)

		if ipFamily == corev1.IPv4Protocol && ip.To4() != nil {
			return ipStr
		}

		if ipFamily == corev1.IPv6Protocol && ip.To4() == nil {
			return ipStr
		}
	}

	return "<no IP for " + string(ipFamily) + ">"
}

func getNicIPs(pod *corev1.Pod, ifcName string) ([]string, error) {

	networksStatus, ok := pod.ObjectMeta.Annotations[nadv1.NetworkStatusAnnot]
	if !ok {
		return nil, fmt.Errorf("cannot get networks status from pod [%s] annotation [%s]", nadv1.NetworkStatusAnnot, pod.Name)
	}

	var nets []nadv1.NetworkStatus
	err := json.Unmarshal([]byte(networksStatus), &nets)
	if err != nil {
		return nil, err
	}

	for _, net := range nets {
		if net.Interface != ifcName {
			continue
		}
		return net.IPs, nil
	}

	return nil, fmt.Errorf("no IP addresses found for interface [%s], pod [%s]", ifcName, pod.Name)
}

func findContainerNameByImage(pod *corev1.Pod, image string) (string, error) {
	for _, c := range pod.Spec.Containers {
		if c.Image == image {
			return c.Name, nil
		}
	}

	return "", fmt.Errorf("can't find container with image [%s] in pod [%s]", image, pod.Name)
}

// makeConnectivityMatrix returns a string representation of the connectivity matrix between
// specified pods. The following is a sample output:
//
//	Reachability matrix of 9 pods on UDP:5555 (X = true, . = false)
//	      x/a     x/b     x/c     y/a     y/b     y/c     z/a     z/b     z/c
//	x/a   .       X       X       X       X       X       X       X       X
//	x/b   X       .       X       X       X       X       X       X       X
//	x/c   X       X       .       X       X       X       X       X       X
//	y/a   X       X       X       .       X       X       X       X       X
//	y/b   X       X       X       X       .       X       X       X       X
//	y/c   X       X       X       X       X       .       X       X       X
//	z/a   X       X       X       X       X       X       .       X       X
//	z/b   X       X       X       X       X       X       X       .       X
//	z/c   X       X       X       X       X       X       X       X       .
func makeConnectivityMatrix(destinationPort string, ipFamily corev1.IPFamily, protocol corev1.Protocol, pods ...*corev1.Pod) string {

	type connectivityPair struct {
		from  *corev1.Pod
		to    *corev1.Pod
		value bool
	}

	data := make(chan connectivityPair, 81)

	connectivityMatrix := make(map[*corev1.Pod]map[*corev1.Pod]bool)

	var conversionWG sync.WaitGroup
	conversionWG.Add(1)
	go func() {
		defer conversionWG.Done()
		for k := range data {
			from, ok := connectivityMatrix[k.from]
			if !ok {
				from = make(map[*corev1.Pod]bool)
				connectivityMatrix[k.from] = from
			}
			from[k.to] = k.value
		}
	}()

	var producerWG sync.WaitGroup

	for _, source := range pods {
		for _, destination := range pods {
			producerWG.Add(1)
			d := destination
			s := source
			go func() {
				defer producerWG.Done()

				if s == nil || d == nil {
					return
				}
				canReach, err := canSendTraffic(s, d, destinationPort, ipFamily, protocol)
				if err != nil {
					fmt.Println(err.Error())
					return
				}

				data <- connectivityPair{from: s, to: d, value: canReach}
			}()
		}
	}

	producerWG.Wait()
	close(data)
	conversionWG.Wait()

	return convertToString(destinationPort, ipFamily, protocol, connectivityMatrix, pods...)
}

func convertToString(
	destinationPort string,
	ipFamily corev1.IPFamily,
	protocol corev1.Protocol,
	connectivityMatrix map[*corev1.Pod]map[*corev1.Pod]bool,
	pods ...*corev1.Pod) string {

	ret := fmt.Sprintf("Reachability matrix of %d pods on %s, %s:%s (X = true, . = false)\n",
		len(pods), ipFamily, protocol, destinationPort)

	for _, destination := range pods {
		ret += fmt.Sprintf("\t%s", shortName(destination))
	}
	ret += "\n"

	for _, source := range pods {
		ret += shortName(source)
		from, ok := connectivityMatrix[source]
		if !ok {
			ret += "\n"
			continue
		}
		for _, destination := range pods {
			ret += "\t"
			canReach, ok := from[destination]
			if !ok {
				ret += "?"
				continue
			}
			if canReach {
				ret += "X"
				continue
			}

			ret += "."
		}
		ret += "\n"
	}

	return ret
}

func shortName(p *corev1.Pod) string {
	ns := ""
	switch p.Namespace {
	case nsX:
		ns = "x"
	case nsY:
		ns = "y"
	case nsZ:
		ns = "z"
	}

	podLabel, ok := p.ObjectMeta.Labels["pod"]
	if !ok {
		podLabel = "?"
	}
	return ns + "/" + podLabel
}
