
<!--- IMPORTANT!
This file is generated manually. To add a new description please run
hack/fill-empty-docs.sh, check the json description files and fill the missing descriptions (the placeholder is XXXXXX)
--->

# Validation Test List

The validation tests are preliminary tests intended to verify that the instrumented features are available on the cluster.
| Test Name | Description |
| -- | ----------- |
| validation container-mount-namespace should have a container-mount-namespace machine config for master | Check the presence of a machine config that enables container-mount-namespace on masters. | 
| validation container-mount-namespace should have a container-mount-namespace machine config for worker | Check the presence of a machine config that enables container-mount-namespace on workers. | 
| validation container-mount-namespace should have the container-mount-namespace machine config as part of the master machine config pool | Check if the container-mount-namespace machine config is used by the master machine config pool | 
| validation container-mount-namespace should have the container-mount-namespace machine config as part of the worker machine config pool | Check if the container-mount-namespace machine config is used by the worker machine config pool | 
| validation dpdk should have a tag ready from the dpdk imagestream | Check that, if a dpdk imagestream exists, it also have a tag ready to be consumed | 
| validation fec should have a ready deployment for the OpenNESS Operator for Intel FPGA PAC N3000 (Management) operator | Checks Intel FPGA PAC N3000 (Management) deployment ready - sriov-fec-controller-manager | 
| validation fec should have all the required OpenNESS Operator for Intel FPGA PAC N3000 (Management) operands | Checks the existence and quantity of each Intel FPGA PAC N3000 (Management) daemonset | 
| validation fec should have the fec CRDs available in the cluster | Checks the existence of the Intel FPGA PAC N3000 (Management) CRDs used by the Intel FPGA PAC N3000 (Management) operator. | 
| validation gatekeeper mutation should have the gatekeeper namespace | Checks the existence of the gatekeeper namespace. | 
| validation gatekeeper mutation should have the gatekeeper-audit deployment in running state | Checks that all the audit deployment pods are running. | 
| validation gatekeeper mutation should have the gatekeeper-controller-manager deployment in running state | Checks that all the mutation deployment pods are running. | 
| validation gatekeeper mutation should have the gatekeeper-operator-controller-manager deployment in running state | Checks that all the operator deployment pods are running. | 
| validation general [ovn] should have a openshift-ovn-kubernetes namespace | Checks the presence of the ovn-k8s namespace, to make sure that the ovn-k8s is used as sdn. | 
| validation general should have all the nodes in ready | Checks that all the nodes are in ready state | 
| validation general should have one machine config pool with the requested label | Checks the existance of a machine config pool with the value passed in the ROLE_WORKER_CNF env variable (or the default worker-cnf). | 
| validation general should report all machine config pools are in ready status | Checks that all the machine config pools are ready so the tests can be run. | 
| validation n3000 should have a ready deployment for the OpenNESS Operator for Intel FPGA PAC N3000 (Programming) operator | Checks Intel FPGA PAC N3000 (Programming) deployment ready - n3000-controller-manager | 
| validation n3000 should have all the required OpenNESS Operator for Intel FPGA PAC N3000 (Programming) operands | Checks the existence and quantity of each Intel FPGA PAC N3000 (Programming) daemonset | 
| validation n3000 should have the n3000 CRDs available in the cluster | Checks the existence of the Intel FPGA PAC N3000 (Programming) CRDs used by the Intel FPGA PAC N3000 (Programming) operator. | 
| validation performance Should have the performance CRD available in the cluster | Checks the existence of the PerformanceProfile CRD used by the Performance Addon Operator. | 
| validation performance should have the performance operator deployment in running state | Check if the Performance Addon Operator is running. | 
| validation performance should have the performance operator namespace | Checks the existence of the Performance Addon Operator's namespace. | 
| validation ptp should have the linuxptp daemonset in running state | Check if the linuxptp daemonset is running. | 
| validation ptp should have the ptp CRDs available in the cluster | Checks the existence of the ptp CRDs used by the PTP Operator. | 
| validation ptp should have the ptp namespace | Checks the existence of the PTP Operator's namespace. | 
| validation ptp should have the ptp operator deployment in running state | Check if the PTP Operator is running. | 
| validation sctp should have a sctp enable machine config | Check the presence of a machine config that enables sctp. | 
| validation sctp should have the sctp enable machine config as part of the CNF machine config pool | Check if the sctp machine config is used by the declared machine config pool. | 
| validation sriov Should have the sriov CRDs available in the cluster | Checks the existence of the SR-IOV CRDs used by the SR-IOV Operator. | 
| validation sriov should deploy the injector pod if requested | Check the optional presence of the SR-IOV injector pod | 
| validation sriov should deploy the operator webhook if requested | Check the optional presence of the SR-IOV webhook | 
| validation sriov should have SR-IOV node statuses not in progress | Check that all the SR-IOV node state resources are not in progress | 
| validation sriov should have the sriov namespace | Checks the existence of the SR-IOV Operator's namespace. | 
| validation sriov should have the sriov operator deployment in running state | Check if the SR-IOV Operator is running. | 
| validation xt_u32 should have a xt_u32 enable machine config | Check the presence of a machine config that enables xt_u32. | 
| validation xt_u32 should have the xt_u32 enable machine config as part of the CNF machine config pool | Check if the xt_u32 machine config is used by the declared machine config pool. | 

# CNF Tests List
The cnf tests instrument each different feature required by CNF. Following, a detailed description for each test.


## DPDK

| Test Name | Description |
| -- | ----------- |
| dpdk Downward API Volume is readable in container | Verifies that the container downward API volumes are readable and match the Pod inforamtion details | 
| dpdk VFS allocated for dpdk Validate HugePages SeLinux access should allow to remove the hugepage file inside the pod | Verifies that we are able to remove the hugepages files after the DPDK running | 
| dpdk VFS allocated for dpdk Validate HugePages should allocate the amount of hugepages requested | Verifies that the number of hugepages requested by the pod are allocated. | 
| dpdk VFS allocated for dpdk Validate NUMA aliment should allocate all the resources on the same NUMA node | Verifies that both the cpus and the pci resources are allocated to the same numa node. | 
| dpdk VFS allocated for dpdk Validate NUMA aliment should allocate the requested number of cpus | Verifies that the number of requested CPUs are allocated to a pod. | 
| dpdk VFS allocated for dpdk Validate a DPDK workload running inside a pod Should forward and receive packets | Verifies that the testpmd application inside a pod is able to receive and send packets. | 
| dpdk VFS allocated for dpdk Validate the build Should forward and receive packets from a pod running dpdk base on a image created by building config | Verifies that the testpmd application inside a pod is able to receive and send packets using an image built via the build pipeline. | 
| dpdk VFS split for dpdk and netdevice Run a regular pod using a vf shared with the dpdk's pf | Verifies that a regular pod can run while sharing vfs with a pod using a vf for dpdk payload | 
| dpdk VFS split for dpdk and netdevice should forward and receive packets from a pod running dpdk base | Verifies that the testpmd application inside a pod is able to receive and send packets, when the pf is shared between regular netdevice pods and dpdk pods. | 
| dpdk dpdk application on different vendors Test connectivity using the requested nic Ethernet Controller XXV710 Intel(R) FPGA Programmable Acceleration Card N3000 for Networking | Verifies dpdk works on Ethernet Controller XXV710 Intel(R) FPGA Programmable Acceleration Card N3000 nic | 
| dpdk dpdk application on different vendors Test connectivity using the requested nic Ethernet controller: Mellanox Technologies MT27710 Family [ConnectX-4 Lx] | Verifies dpdk works on Ethernet controller: Mellanox Technologies MT27710 Family [ConnectX-4 Lx] nic | 
| dpdk dpdk application on different vendors Test connectivity using the requested nic Ethernet controller: Mellanox Technologies MT27800 Family [ConnectX-5] | Verifies dpdk works on Ethernet controller: Mellanox Technologies MT27800 Family [ConnectX-5] nic | 
| dpdk dpdk application on different vendors Test connectivity using the requested nic Intel Corporation Ethernet Controller XXV710 for 25GbE SFP28 | Verifies dpdk works on Intel Corporation Ethernet Controller XXV710 for 25GbE SFP28 nic | 
| dpdk restoring configuration should restore the cluster to the original status | Verifies that the cluster state is restored after running the dpdk tests. | 

## SR-IOV

| Test Name | Description |
| -- | ----------- |
| [sriov] SCTP integration Test Connectivity Connectivity between client and server Should work over a SR-IOV device | SCTP connectivity test over SR-IOV vfs. | 
| [sriov] VRF integration  Integration: SRIOV, IPAM: static, Interfaces: 1, Scheme: 2 Pods 2 VRFs OCP Primary network overlap {"IPStack":"ipv4"} | Verifies that it's possible to configure within the same node 1 VRF that overlaps pod's network + 2 non overlapping VRF on top of SriovNetwork. Connectivity ICMP test. | 
| [sriov] operator Custom SriovNetworkNodePolicy Configuration Create vfio-pci node policy Should be possible to create a vfio-pci resource | Verifies creating of vfio-pci resources | 
| [sriov] operator Custom SriovNetworkNodePolicy Configuration MTU Should support jumbo frames | SR-IOV connectivity tests with jumbo frames. | 
| [sriov] operator Custom SriovNetworkNodePolicy Configuration Main PF should work when vfs are used by pods | Verifies that it's possible to use the PF as a network interface with VFs are used by pod workloads | 
| [sriov] operator Custom SriovNetworkNodePolicy Configuration PF Partitioning Should be possible to partition the pf's vfs | Verifies that it's possible to partition the vfs associated to a given vf with different configurations. | 
| [sriov] operator Custom SriovNetworkNodePolicy Configuration PF Partitioning Should not be possible to have overlapping pf ranges | Verifies creating of overlapping pf ranges | 
| [sriov] operator Custom SriovNetworkNodePolicy Configuration PF shutdown Should be able to create pods successfully if PF is down.Pods are able to communicate with each other on the same node | Checks that the pods are able to use the vfs even if the pf is down. | 
| [sriov] operator Custom SriovNetworkNodePolicy Nic Validation Test connectivity using the requested nic Ethernet Controller XXV710 Intel(R) FPGA Programmable Acceleration Card N3000 for Networking | Optional test to assert that N3000 works for networking | 
| [sriov] operator Custom SriovNetworkNodePolicy Nic Validation Test connectivity using the requested nic Intel Corporation Ethernet Controller XXV710 for 25GbE SFP28 | Optional test to assert that 25GbE SFP28 works for networking | 
| [sriov] operator Custom SriovNetworkNodePolicy Resource Injector SR-IOV Operator Config, disable Network resource injector | Verifies that by disabling the network resource injector in the config, the injector is really disabled. | 
| [sriov] operator Custom SriovNetworkNodePolicy Resource Injector SR-IOV Operator Config, disable Webhook resource injector | Verifies that by disabling the mutating webhook in the config, the webhook is really disabled. | 
| [sriov] operator Generic SriovNetworkNodePolicy IPv6 configured secondary interfaces on pods should be able to ping each other | Connectivity test via icmp for two ipv6 configured interfaces. | 
| [sriov] operator Generic SriovNetworkNodePolicy Meta Plugin Configuration Should be able to configure a metaplugin | Verifies that it's possible to configure a metaplugin in chain with the sriov CNI plugin | 
| [sriov] operator Generic SriovNetworkNodePolicy Multiple sriov device and attachment Should configure multiple network attachments | Checks that when adding multiple networks to the pod, multiple interfaces are created inside the pod. | 
| [sriov] operator Generic SriovNetworkNodePolicy NAD update NAD default gateway is updated when SriovNetwork ipam is changed | Checks that the network attachment definition name is updated when the SriovNetwork ipam section is changed. | 
| [sriov] operator Generic SriovNetworkNodePolicy NAD update NAD is updated when SriovNetwork spec/networkNamespace is changed | Checks that the network attachment definition name is updated when the SriovNetwork namespace / specs are changed. | 
| [sriov] operator Generic SriovNetworkNodePolicy Resource Injector Should inject downward api volume with labels present | Checks the downward api volume is present when labels are added | 
| [sriov] operator Generic SriovNetworkNodePolicy Resource Injector Should inject downward api volume with no labels present | Checks the downward api volume is present when no labels are added | 
| [sriov] operator Generic SriovNetworkNodePolicy SRIOV and macvlan Should be able to create a pod with both sriov and macvlan interfaces | Verifies that it's possible to create a pod with both SR-IOV and MACVlan interfaces. | 
| [sriov] operator Generic SriovNetworkNodePolicy VF flags Should configure the spoofChk boolean variable | Verifies that a vf can be configured with the spoofCheck variable. | 
| [sriov] operator Generic SriovNetworkNodePolicy VF flags Should configure the the link state variable | Verifies that the configuration is able to set the link state of a VF. | 
| [sriov] operator Generic SriovNetworkNodePolicy VF flags Should configure the trust boolean variable | Verifies that it's possible to set the trust variable on a vf via configuration. | 
| [sriov] operator Generic SriovNetworkNodePolicy VF flags rate limit Should configure the requested rate limit flags under the vf | Verifies that it's possible to set the rate limiting of a given VF. | 
| [sriov] operator Generic SriovNetworkNodePolicy VF flags vlan and Qos vlan Should configure the requested vlan and Qos vlan flags under the vf | Verifies that it's possible to set vlan and QoS flags to a given VF. | 
| [sriov] operator Generic SriovNetworkNodePolicy Virtual Functions should release the VFs once the pod deleted and same VFs can be used by the new created pods | Verifies that an allocated VF is released when the pod that was using it is deleted. | 
| [sriov] operator No SriovNetworkNodePolicy SR-IOV network config daemon can be set by nodeselector Should schedule the config daemon on selected nodes | Verifies that it's possible to configure | 

## SCTP

| Test Name | Description |
| -- | ----------- |
| sctp Negative - Sctp disabled Client Server Connection Should NOT start a server pod | Negative test: when the sctp module is not enabled, verifies that the connectivity is not working. | 
| sctp Test Connectivity Connectivity between client and server Connectivity Test Custom namespace | Pod to pod connectivity within a custom namespace. | 
| sctp Test Connectivity Connectivity between client and server Connectivity Test Custom namespace with policy | Verifies that the connectivity works when putting a matching network policy. | 
| sctp Test Connectivity Connectivity between client and server Connectivity Test Custom namespace with policy no port | Verifies that a blocking network policy stops the connectivity. | 
| sctp Test Connectivity Connectivity between client and server Connectivity Test Default namespace | Pod to pod connectivity, default namespace. | 
| sctp Test Connectivity Connectivity between client and server Connectivity Test Default namespace with policy | Verifies that the connectivity works when putting a matching network policy, default namespace. | 
| sctp Test Connectivity Connectivity between client and server Connectivity Test Default namespace with policy no port | Verifies that a blocking network policy stops the connectivity, default namespace. | 
| sctp Test Connectivity Connectivity between client and server Kernel Module is loaded | Check that the kernel module is loaded | 
| sctp Test Connectivity Connectivity between client and server Should connect a client pod to a server pod. Feature LatencySensitive Active | Check that pod 2 pod connectivity works with the LatencySensitive Feature activated | 
| sctp Test Connectivity Connectivity between client and server connect a client pod to a server pod via Service ClusterIP Custom namespace | Pod to pod connectivity via service ClusterIP, custom namespace | 
| sctp Test Connectivity Connectivity between client and server connect a client pod to a server pod via Service ClusterIP Default namespace | Pod to pod connectivity via service ClusterIP, default namespace | 
| sctp Test Connectivity Connectivity between client and server connect a client pod to a server pod via Service Node Port Custom namespace | Pod to pod connectivity via service nodeport, custom namespace | 
| sctp Test Connectivity Connectivity between client and server connect a client pod to a server pod via Service Node Port Default namespace | Pod to pod connectivity via service nodeport, default namespace | 

## Performance

| Test Name | Description |
| -- | ----------- |
| [performance] Additional kernel arguments added from perfomance profile  Should set additional kernel arguments on the machine | Verifies that when specifying additional kernel arguments to the profile, those are added on the node. | 
| [performance] CPU Management Verification of configuration on the worker node  Verify CPU affinity mask, CPU reservation and CPU isolation on worker node | Verifies that CPU affinity, reservation and isolation are set correctly on the node as specified in the profile spec. | 
| [performance] CPU Management Verification of configuration on the worker node  Verify CPU reservation on the node | When specifying reserved CPUs, verifies that they don't belong to the allocatable list. | 
| [performance] CPU Management Verification of configuration on the worker node  Verify rcu_nocbs kernel argument on the node | Checks that the node has rcu_nocbs kernel argument applied | 
| [performance] CPU Management Verification of cpu manager functionality Verify CPU usage by stress PODs  Guaranteed POD should work on isolated cpu | Checks that the guaranteed pod will use the isolated CPU, the test relevant only for cases when reserved and isolated CPUs complementary and include all online CPUs | 
| [performance] CPU Management Verification of cpu manager functionality Verify CPU usage by stress PODs  Non-guaranteed POD can work on any CPU | Checks that non guaranteed pod can use any CPU | 
| [performance] CPU Management Verification that IRQ load balance can be disabled per POD  should disable IRQ balance for CPU where POD is running | Checks that the runtime will disable the IRQ load balancing for CPUs used by the guaranteed pod, when the pod has the specific runtime class and annotation | 
| [performance] CPU Management when pod runs with the CPU load balancing runtime class  should disable CPU load balancing for CPU's used by the pod | Checks that the runtime will disable the CPU load balancing for the guaranteed pod with the specific runtime class and annotation | 
| [performance] CPU Management when reserved CPUs specified should run infra containers on reserved CPUs | Checks that infra containers runs on top of reserved CPUs | 
| [performance] Create second performance profiles on a cluster  Verifies that cluster can have multiple profiles | Verifies that multiple performance profiles can be applied to the cluster. | 
| [performance] KubeletConfig experimental annotation should override system-reserved memory | Verifies that KubeletConfig experimental annotation should override system-reserved memory | 
| [performance] Latency Test with the oslat image should succeed | Run the oslat with parameters specified via environment variables and validated that the maximum latency for isolated CPUs below the value specified under the OSLAT_MAXIMUM_LATENCY environment variable | 
| [performance] Network latency parameters adjusted by the Node Tuning Operator  Should contain configuration injected through the openshift-node-performance profile | Checks that the node has injected tuned sysctl parameters | 
| [performance] Performance Operator  Should run on the control plane nodes | Verifies that the performance addons operator pod is running on control plane node | 
| [performance] Pre boot tuning adjusted by tuned   Should set CPU affinity kernel argument | Checks that the node has injected systemd.cpu_affinity argument under boot parameters, that used to configure the CPU affinity | 
| [performance] Pre boot tuning adjusted by tuned   Should set CPU isolcpu's kernel argument managed_irq flag | Verifies that the isolcpus kernel argument is set with managed_irq | 
| [performance] Pre boot tuning adjusted by tuned   Should set workqueue CPU mask | Checks that the node has injected workqueue CPU mask | 
| [performance] Pre boot tuning adjusted by tuned   Stalld runs in higher priority than ksoftirq and rcu{c,b} | Verifies that stalld process ir running with a higher priority than ksoftirq and rcu{c,b} | 
| [performance] Pre boot tuning adjusted by tuned   initramfs should not have injected configuration | Checks that the iniramfs does not have injected configuration | 
| [performance] Pre boot tuning adjusted by tuned   stalld daemon is running as sched_fifo | Verifies that stalld daemon has a scheduling policy of SCHED_FIFO with high priority | 
| [performance] Pre boot tuning adjusted by tuned   stalld daemon is running on the host | Checks that the stalld daemon is running on the host | 
| [performance] RPS configuration Should have the correct RPS configuration | Validates that old and newly created vnics should have the RPS mask that excludes CPUs used by guaranteed pod | 
| [performance] Tuned CRs generated from profile  Node should point to right tuned profile | Validates that the active tuned profile under the node should point to the tuned profile generate by the performance-addon-operator | 
| [performance] Tuned CRs generated from profile  Should have the expected name for tuned from the profile owner object | Checks that the PAO generates the tuned resources with the expected name | 
| [performance] Tuned kernel parameters  Should contain configuration injected through openshift-node-performance profile | Checks that the node has kernel arguments that should be injected via tuned | 
| [performance] Validation webhook with API version v1 profile should reject the creation of the profile with no isolated CPUs | Checks that performance profile with API version v1 and without isolated CPUs should be rejected by validation | 
| [performance] Validation webhook with API version v1 profile should reject the creation of the profile with overlapping CPUs | Checks that performance profile with API version v1 and with isolated and reserved overlapping CPUs should be rejected by validation | 
| [performance] Validation webhook with API version v1 profile should reject the creation of the profile with the node selector that already in use | Checks that performance profile with API version v1 and with node selector that already in use by existing performance profile should be rejected by validation | 
| [performance] Validation webhook with API version v1alpha1 profile should reject the creation of the profile with no isolated CPUs | Checks that performance profile with API version v1alpha1 and without isolated CPUs will be rejected by validation | 
| [performance] Validation webhook with API version v1alpha1 profile should reject the creation of the profile with overlapping CPUs | Checks that performance profile with API version v1alpha1 and with isolated and reserved overlapping CPUs should be rejected by validation | 
| [performance] Validation webhook with API version v1alpha1 profile should reject the creation of the profile with the node selector that already in use | Checks that performance profile with API version v1alpha1 and with node selector that already in use by existing performance profile should be rejected by validation | 
| [performance] Validation webhook with profile version v2 should reject the creation of the profile with no isolated CPUs | Checks that performance profile with API version v2 and without isolated CPUs will be rejected by validation | 
| [performance] Validation webhook with profile version v2 should reject the creation of the profile with overlapping CPUs | Checks that performance profile with API version v2 and with isolated and reserved overlapping CPUs should be rejected by validation | 
| [performance] Validation webhook with profile version v2 should reject the creation of the profile with the node selector that already in use | Checks that performance profile with API version v2 and with node selector that already in use by existing performance profile should be rejected by validation | 
| [performance] Verify API Conversions  Verifies v1 <-> v1alpha1 conversions | Verifies that Performance Addon Operator can work with v1alpha1 Performance Profiles. | 
| [performance] Verify API Conversions  Verifies v1 <-> v2 conversions | Verifies that Performance Addon Operator can work with v1 Performance Profiles. | 
| [performance] Verify API Conversions when the performance profile does not contain NUMA field Verifies v1 <-> v1alpha1 conversions | Checks that conversion webhooks succeeds to convert v1 <-> v1alpha1 profiles without NUMA field | 
| [performance] Verify API Conversions when the performance profile does not contain NUMA field Verifies v1 <-> v2 conversions | Checks that conversion webhooks succeeds to convert v1 <-> v2 profiles without NUMA field | 
| [performance]Hugepages Huge pages support for container workloads  Huge pages support for container workloads | Verifies that huge pages are available in a container when requested. | 
| [performance]Hugepages when NUMA node specified  should be allocated on the specifed NUMA node  | Verifies that when hugepages are specified on a given numa node in the profile are allocated to that node. | 
| [performance]Hugepages with multiple sizes  should be supported and available for the container usage | Verifies that hugepages with different size can be configured and used by pods. | 
| [performance]RT Kernel  a node without performance profile applied should not have RT kernel installed | Verifies that RT kernel is not enabled when not configured in the profile. | 
| [performance]RT Kernel  should have RT kernel enabled | Verifies that RT kernel is enabled when configured in the profile. | 
| [performance]Topology Manager  should be enabled with the policy specified in profile | Verifies that when specifying a topology policy in the profile, that is used by the topology manager. | 
| [ref_id: 40307][pao]Resizing Network Queues Updating performance profile for netqueues  Add interfaceName and verify the interface netqueues are equal to reserved cpus count. | Adds interfaceName and verifies the interface netqueues are equal to reserved cpus count | 
| [ref_id: 40307][pao]Resizing Network Queues Updating performance profile for netqueues  Network device queues Should be set to the profile's reserved CPUs count  | Checks that Network device queues Should be set to the profile's reserved CPUs count | 
| [ref_id: 40307][pao]Resizing Network Queues Updating performance profile for netqueues  Verify reserved cpu count is added to networking devices matched with vendor and Device id | Verifies that reserved cpu count is added to networking devices matched with vendor and Device id  | 
| [ref_id: 40307][pao]Resizing Network Queues Updating performance profile for netqueues  Verify reserved cpus count is applied to specific supported networking devices using wildcard matches | Checks that reserved cpus count is applied to specific supported networking devices using wildcard matches | 
| [ref_id: 40307][pao]Resizing Network Queues Updating performance profile for netqueues  Verify the number of network queues of all supported network interfaces are equal to reserved cpus count | Checks that the number of network queues of all supported network interfaces are equal to reserved cpus count | 

## PTP

| Test Name | Description |
| -- | ----------- |
| [ptp] PTP configuration verifications Should check that all nodes are running at least one replica of linuxptp-daemon | Checks if the linuxptp-daemon is running on all the nodes. | 
| [ptp] PTP configuration verifications Should check that operator is deployed | Checks if the ptp operator is deployed. | 
| [ptp] PTP configuration verifications Should check whether PTP operator appropriate resource exists | Checks if the ptp operator CRDs exist on the cluster. | 
| [ptp] PTP e2e tests PTP Interfaces discovery Can provide a profile with higher priority | Checks if when applying a profile with higher priority then it is used. | 
| [ptp] PTP e2e tests PTP Interfaces discovery PTP daemon apply match rule based on nodeLabel | Checks if the ptp daemon applies the correct profile based on the node labels. | 
| [ptp] PTP e2e tests PTP Interfaces discovery Slave can sync to master | Checks if the ptp slave syncs with the master. | 
| [ptp] PTP e2e tests PTP Interfaces discovery The interfaces support ptp can be discovered correctly | Checks if the interfaces supporting ptp are discovered correctly. | 
| [ptp] PTP e2e tests PTP Interfaces discovery The virtual interfaces should be not discovered by ptp | Checks that the virtual interfaces are not used by the ptp daemon | 
| [ptp] PTP e2e tests PTP metric is present on slave | Checks that the metrics related to ptp are produced by the slave. | 
| ptp PTP socket sharing between pods Negative - run pmc in a new unprivileged pod on the slave node Should not be able to use the uds | Verifies that ptp uds socket cannot be used by an unprivileged pod on the slave node | 
| ptp PTP socket sharing between pods Run pmc in a new pod on the slave node Should be able to sync using a uds | Verifies that ptp uds socket is shared between pods on the slave node | 
| ptp Test Offset PTP configuration verifications PTP time diff between Grandmaster and Slave should be in range -100ms and 100ms | Verifies that the time diff between master & slave is below 100 ms. | 
| ptp prometheus Metrics reported by PTP pods Should all be reported by prometheus | Verifies that the PTP metrics are reported. | 

## Others

| Test Name | Description |
| -- | ----------- |
| [vrf]  Integration: NAD, IPAM: static, Interfaces: 1, Scheme: 2 Pods 2 VRFs OCP Primary network overlap {"IPStack":"ipv4"} | Verifies that it's possible to configure within the same node 1 VRF that overlaps pod's network + 2 non overlapping VRF on top of mac-vlan cni which is based on top of default route node's interface. Connectivity ICMP test. | 
| fec Expose resource on the node should show resources under the node | Verifies that the sriov-fec operator is able to create and expose virtual functions from the acc100 accelerator card | 
| gatekeeper mutation should apply mutations by order | Verifies that gatekeeper mutations are applied by order | 
| gatekeeper mutation should avoid mutating existing metadata info(labels/annotations) | Verifies that gatekeeper will not mutate an objects label/annotation if it already exists | 
| gatekeeper mutation should be able to add metadata info(labels/annotations) | Verifies that gatekeeper is able to mutate an object by adding a label/annotation to it | 
| gatekeeper mutation should be able to match by any match category | Verifies that gatekeeper is able to match objects by mutation policy matching categories | 
| gatekeeper mutation should be able to update mutation policy | Verifies that gatekeeper mutation policy can be updated and apply the updated mutation | 
| gatekeeper mutation should not apply mutations policies after deletion | Verifies that gatekeeper will not apply mutations from a deleted mutation policy | 
| gatekeeper operator should be able to select mutation namespaces | Verifies that gatekeeper operator is able to select mutation enabled namespaces | 
| ovs_qos ovs_qos_egress validate egress QoS limitation Test limitations are correctly applied {"Connectivity":"Host Pod to Host Pod"} | Test egress limitation between 2 pods on the hostNetwork | 
| ovs_qos ovs_qos_egress validate egress QoS limitation Test limitations are correctly applied {"Connectivity":"Host Pod to SDN Pod"} | Test egress limitation between a hostNetwork pod and an SDN pod | 
| ovs_qos ovs_qos_egress validate egress QoS limitation Test limitations are correctly applied {"Connectivity":"SDN Pod to SDN Pod"} | Test egress limitation between 2 SDN pods | 
| ovs_qos ovs_qos_egress validate egress QoS limitation Test limitations are not applied within the same node {"Connectivity":"Host Pod to SDN Pod"} | Test egress limitation between a hostNetwork pod and an SDN pod on the same node does not limit | 
| ovs_qos ovs_qos_egress validate egress QoS limitation Test limitations are not applied within the same node {"Connectivity":"SDN Pod to SDN Pod"} | Test egress limitation between a SDN pod and an SDN pod on the same node does not limit | 
| ovs_qos ovs_qos_egress validate egress QoS limitation Validate MCO applied egress MachineConfig on the relevant nodes | Validate that the egress MachineConfig is applied correctly and present in the ovs database | 
| ovs_qos ovs_qos_egress validate egress QoS limitation Validate MCO removed egress MachineConfig and disabled QOS limitation on the relevant nodes | Validate that egress MachineConfig was removed correctly and QoS removed from ovs port | 
| ovs_qos ovs_qos_ingress validate ingress QoS limitation Test limitations are correctly applied {"Connectivity":"Host Pod to Host Pod"} | Test ingress limitation between 2 pods on the hostNetwork | 
| ovs_qos ovs_qos_ingress validate ingress QoS limitation Test limitations are correctly applied {"Connectivity":"Host Pod to SDN Pod"} | Test ingress limitation between a hostNetwork pod and an SDN pod | 
| ovs_qos ovs_qos_ingress validate ingress QoS limitation Test limitations are correctly applied {"Connectivity":"SDN Pod to SDN Pod"} | Test ingress limitation between 2 SDN pods | 
| ovs_qos ovs_qos_ingress validate ingress QoS limitation Test limitations are not applied within the same node {"Connectivity":"Host Pod to SDN Pod"} | Test ingress limitation between a hostNetwork pod and an SDN pod on the same node does not limit | 
| ovs_qos ovs_qos_ingress validate ingress QoS limitation Test limitations are not applied within the same node {"Connectivity":"SDN Pod to SDN Pod"} | Test ingress limitation between a SDN pod and an SDN pod on the same node does not limit | 
| ovs_qos ovs_qos_ingress validate ingress QoS limitation Validate MCO applied ingress MachineConfig on the relevant nodes | Validate that the ingress MachineConfig is applied correctly and present in the ovs database | 
| ovs_qos ovs_qos_ingress validate ingress QoS limitation Validate MCO removed ingress MachineConfig and disabled QOS limitation on the relevant nodes | Validate that ingress MachineConfig was removed correctly and QoS removed from ovs interface | 
| xt_u32 Negative - xt_u32 disabled Should NOT create an iptable rule | Negative test: when the xt_u32 module is not enabled, appling an iptables rule that utilize the module should fail. | 
| xt_u32 Validate the module is enabled and works Should create an iptables rule inside a pod that has the module enabled | Verifies that an iptables rule that utilize xt_u32 module can be applied successfully in a pod that has the module enabled. | 

