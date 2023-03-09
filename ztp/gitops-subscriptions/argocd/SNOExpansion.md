# SNO expansion #

SNO expansion is a scenario where initially a SNO is deployed into production, and at a later date the cluster size is increased by the addition of one or more worker nodes.
The resulting cluster will contain at least two nodes:
- Original SNO serving as the cluster control plane
- One or more worker nodes without any control plane components (controlled by the SNO).

This transition incurs zero downtime on the original SNO node.
Addition of workers to an SNO cluster will increase CPU resources used by the original (master) node for control plane functions it performs. Although there is no hard limit on amount of the additional workers that can be added, worker addition must come with re-evaluation of reserved CPU allocation on the master. 

## Prerequisites ##
1. A cluster installed and configured using the GitOps ZTP flow, as described in [GitOps ZTP flow](README.md)
1. ACM 2.6 (or above) with MultiClusterHub created and configured, running on OCP 4.11 (or above) bare metal cluster
1. Central Infrastructure Management configured as described in [Advanced Cluster Management documentation](https://access.redhat.com/documentation/en-us/red_hat_advanced_cluster_management_for_kubernetes/2.5/html/clusters/managing-your-clusters#enable-cim)
1. TALM and openshift-gitops operators are installed and configured on the hub cluster
1. The DNS serving the cluster is configured to properly resolve internal API endpoint `api-int.<cluster_name>.<base_domain>`. Please check [Requirements for installing OpenShift on a single node](https://docs.openshift.com/container-platform/4.11/installing/installing_sno/install-sno-preparing-to-install-sno.html#install-sno-requirements-for-installing-on-a-single-node_install-sno-preparing) for more details.
1. ZTP container version is 4.12 or above
1. Initial state:
   1. The spoke cluster is managed by the hub.
   2. ArgoCD applications are synchronized and all policies are compliant on the spoke cluster.

## Procedure ##
### Operation order ###
If workload partitioning is required on the worker node, the policies configuring it must be deployed and remediated before the node installation. This way the workload partitioning machineconfig objects will be rendered and associated with the `worker` machineconfig pool before its ignition is downloaded by the installing worker node.
Therefore, the recommended procedure order is the opposite from the SNO installation order: first policies, then installation.
If from any reason workload partitioning manifests are created after the node is already installed, a manual operation is required for draining the node and deleting all the pods managed by daemonsets. The new pods restarted by their managing daemonsets will undergo the workload partitioning process when they are created.


### Applying the DU profile
This procedure assumes the SNO cluster being expanded is provisioned with DU profile, as described in the [README.md](README.md)
The DU profile is provisioned using `PolicyGenTemplate` resources partitioned into common, group and site-specific manifests.
For example, for SNO, the git repository linked to the `policies` ArgoCD application will include:
- [common-ranGen.yaml](example/policygentemplates/common-ranGen.yaml). This template usually contains a set of operator subscriptions and it's unlikely that it should be modified for a worker addition.
- [group-du-sno-ranGen.yaml](example/policygentemplates/group-du-sno-ranGen.yaml)
- [example-sno-site.yaml](example/policygentemplates/example-sno-site.yaml)
- [ns.yaml](example/policygentemplates/ns.yaml)
- [kustomization.yaml](example/policygentemplates/kustomization.yaml)

The procedure of configuring the DU profile on the worker node is considered as an upgrade. To initiate the upgrade flow, user must update the existing policies, or create an additional ones, and then create a ClusterGroupUpgrade to reconcile the policies in the group of clusters.

#### __Preparations (optional - for ZTP DU profile version 4.11 or below)__
If the SNO DU profile was deployed using ZTP plugin version 4.11 or below, PTP and SR-IOV operators might be configured to place the daemons only on nodes labelled as 'master'. This configuration, if exists, would prevent PTP / SR-IOV daemons from operating on the worker node. If PTP / SR-IOV daemon node selectors are incorrectly configured on your system, they must be changed before proceeding with the worker DU profile configuration.
##### __Ensuring PTP / SR-IOV daemon selector compatibility__ 

Check PTP operator `daemonNodeSelector`setting on one of the spoke clusters:
```bash
$ oc get ptpoperatorconfig/default -n openshift-ptp -ojsonpath='{.spec}'
{"daemonNodeSelector":{"node-role.kubernetes.io/master":""}}
```
If the result contains node selector set to "master", as shown above, the spoke was deployed with the version of ztp plugin that requires a special treatment.

In the group policy, make the additions of `complianceType` and `spec` as shown below:
```yaml
spec:
    - fileName: PtpOperatorConfig.yaml
      policyName: "config-policy"
      complianceType: mustonlyhave
      spec:
        daemonNodeSelector:
          node-role.kubernetes.io/worker: ""
    - fileName: SriovOperatorConfig.yaml
      policyName: "config-policy"
      complianceType: mustonlyhave
      spec:
        configDaemonNodeSelector:
          node-role.kubernetes.io/worker: ''
```
When synchronized to the hub, the correspondent policy will become non-compliant. Use TALM operator to apply the changes to your spokes within a maintenance window.

**Important: changing the daemon node selectors will incur temporary PTP synchronization loss / SR-IOV connectivity loss**

##### __Ensuring PTP/SR-IOV configuration node selectors compatibility__
PTP configuration resource and SR-IOV network node policies are using `node-role.kubernetes.io/master: ""` as node selectors. If the additional worker node(s) have the same NIC configuration as the master, the policies used to configure the master can be reused for the workers. The node selector, however, must be changed to select both node types.

1. Label the master and all the additional nodes sharing the same configuration with a common label. For example:
```bash
$ oc label node/<node name> node-role.ran.openshift.io=du
```
2. Modify PTP configuration / SR-IOV network node policies to use the new label for selectors, for example:
**PTP configuration:**
```yaml
    - fileName: PtpConfigSlave.yaml   
      policyName: "config-policy"
      complianceType: mustonlyhave
      metadata:
        name: "du-ptp-slave"
      spec:
        profile:
        - name: "slave"
          # This interface must match the hardware in this group
          interface: "ens1f0"
          ptp4lOpts: "-2 -s --summary_interval -4"
          phc2sysOpts: "-a -r -n 24"
        recommend:
        - match:
          - nodeLabel: node-role.ran.openshift.io=du
          priority: 4
          profile: slave-worker
```   
**SR-IOV network node policies:**
```yaml
    - fileName: SriovNetworkNodePolicy.yaml
      policyName: "config-policy"
      complianceType: mustonlyhave
      metadata:
        name: "sriov-nnp-uplane"
      spec:
        deviceType: vfio-pci
        isRdma: false
        nicSelector:
          pfNames: ["ens1f1"]
        nodeSelector:
          node-role.ran.openshift.io: du
        numVfs: 8
        priority: 10
        resourceName: uplane    
```
When synchronized to the hub, the correspondent policy will become non-compliant. Use TALM operator to apply the changes to your spokes within a maintenance window.

#### __Preparing worker node policies__
Create the following policy template:

```yaml
apiVersion: ran.openshift.io/v1
kind: PolicyGenTemplate
metadata:
  name: "cnfdf15-workers"
  namespace: "ztp-cnfdf15-policies"
spec:
  bindingRules:
    # These policies will correspond to all clusters with this label:
    sites: "cnfdf15"
  # Set MCP to "worker"
  mcp: "worker"
  sourceFiles:
    # This generic MachineConfig CR is used here to configure workload
    # partitioning on the worker node.
    - fileName: MachineConfigGeneric.yaml
      policyName: "config-policy"
      metadata:
        labels:
          machineconfiguration.openshift.io/role: worker
        name: enable-workload-partitioning
      spec:
        config:
          storage:
            files:
            - contents:
                source: data:text/plain;charset=utf-8;base64,W2NyaW8ucnVudGltZS53b3JrbG9hZHMubWFuYWdlbWVudF0KYWN0aXZhdGlvbl9hbm5vdGF0aW9uID0gInRhcmdldC53b3JrbG9hZC5vcGVuc2hpZnQuaW8vbWFuYWdlbWVudCIKYW5ub3RhdGlvbl9wcmVmaXggPSAicmVzb3VyY2VzLndvcmtsb2FkLm9wZW5zaGlmdC5pbyIKcmVzb3VyY2VzID0geyAiY3B1c2hhcmVzIiA9IDAsICJjcHVzZXQiID0gIjAtMyIgfQo=
              mode: 420
              overwrite: true
              path: /etc/crio/crio.conf.d/01-workload-partitioning
              user:
                name: root
            - contents:
                source: data:text/plain;charset=utf-8;base64,ewogICJtYW5hZ2VtZW50IjogewogICAgImNwdXNldCI6ICIwLTMiCiAgfQp9Cg==
              mode: 420
              overwrite: true
              path: /etc/kubernetes/openshift-workload-pinning
              user:
                name: root
    - fileName: PerformanceProfile.yaml
      policyName: "config-policy"
      metadata:
        name: openshift-worker-node-performance-profile
      spec:
        cpu:
          # These must be tailored for the specific hardware platform
          isolated: "4-47"
          reserved: "0-3"
        hugepages:
          defaultHugepagesSize: 1G
          pages:
            - size: 1G
              count: 32
    - fileName: TunedPerformancePatch.yaml
      policyName: "config-policy"
      metadata:
        name: performance-patch-worker
      spec:
        profile:
          - name: performance-patch-worker
            # The cmdline_crash CPU set must match the 'isolated' set in the PerformanceProfile above
            data: |
              [main]
              summary=Configuration changes profile inherited from performance created tuned
              include=openshift-node-performance-openshift-worker-node-performance-profile
              [bootloader]
              cmdline_crash=nohz_full=4-47
              [sysctl]
              kernel.timer_migration=1
              [scheduler]
              group.ice-ptp=0:f:10:*:ice-ptp.*
              [service]
              service.stalld=start,enable
              service.chronyd=stop,disable
        recommend:
        - profile: performance-patch-worker
```
##### __Creating content for workload partitioning machineconfig__
A generic MachineConfig CR is used here to configure workload partitioning on the worker node. The content of `crio` and `kubelet` configuration files can be generated as follows:
- /etc/crio/crio.conf.d/01-workload-partitioning
    ```bash
    $ CPUSET="0-3" # Adjust for your requirements
    $ cat <<EOF | base64 -w 0
    [crio.runtime.workloads.management]
    activation_annotation = "target.workload.openshift.io/management"
    annotation_prefix = "resources.workload.openshift.io"
    resources = { "cpushares" = 0, "cpuset" = "$CPUSET" }
    EOF

    W2NyaW8ucnVudGltZS53b3JrbG9hZHMubWFuYWdlbWVudF0KYWN0aXZhdGlvbl9hbm5vdGF0aW9uID0gInRhcmdldC53b3JrbG9hZC5vcGVuc2hpZnQuaW8vbWFuYWdlbWVudCIKYW5ub3RhdGlvbl9wcmVmaXggPSAicmVzb3VyY2VzLndvcmtsb2FkLm9wZW5zaGlmdC5pbyIKcmVzb3VyY2VzID0geyAiY3B1c2hhcmVzIiA9IDAsICJjcHVzZXQiID0gIjAtMyIgfQo=
    ```

- /etc/kubernetes/openshift-workload-pinning
    ```bash
    cat <<EOF | base64 -w 0
    {
      "management": {
        "cpuset": "$CPUSET"
      }
    }
    EOF

    ewogICJtYW5hZ2VtZW50IjogewogICAgImNwdXNldCI6ICIwLTMiCiAgfQp9Cg==
    ```
#### __Applying the worker node policies__
1. Add the policy template created above to the git repository monitored by the `policies` ArgoCD application. 
2. List the policy in the `kustomization.yaml` file
3. Check in, push and re-sync the `policies` ArgoCD application to apply the generated policies to your hub cluster
#### __Remediating worker node policies__
To remediate the new policies to your spoke cluster, create a new TALM custom resource:

```bash
$ cat <<EOF | oc apply -f -
apiVersion: ran.openshift.io/v1alpha1
kind: ClusterGroupUpgrade
metadata:
  name: cnfdf15-worker-policies
  namespace: default
spec:
  backup: false
  clusters:
  - cnfdf15
  enable: true
  managedPolicies:
  - cnfdf15-worker-config-policy
  preCaching: false
  remediationStrategy:
    maxConcurrency: 1
EOF
```

### Deploying a worker node ###
1. Assuming your cluster was deployed using [this SiteConfig manifest](example/siteconfig/example-sno.yaml), add your new worker node to `spec.clusters['example-sno'].nodes` list, for example:

```yaml
      nodes:
      - hostName: "example-node2.example.com"
        role: "worker"
        bmcAddress: "<BMC address>"
        bmcCredentialsName:
          name: "example-node2-bmh-secret"
        bootMACAddress: "<MAC of the machine network interface>"
        bootMode: "UEFI"
        rootDeviceHints:
          deviceName: "/dev/<device>"
        nodeNetwork:
          interfaces:
            - name: eno1
              macAddress: "<MAC of the machine network interface>"
          config:
            interfaces:
              - name: eno1
                type: ethernet
                state: up
                ipv4:
                  enabled: false
                ipv6:
                  enabled: true
                  address:
                  - ip: 1111:2222:3333:4444::aaaa:2
                    prefix-length: 64
            dns-resolver:
              config:
                search:
                - example.com
                server:
                - 1111:2222:3333:4444::2
            routes:
              config:
              - destination: ::/0
                next-hop-interface: eno1
                next-hop-address: 1111:2222:3333:4444::1
                table-id: 254

```
2. Create secret with BMC credentials for the new host, as referenced by `bmcCredentialsName` in the SiteConfig `nodes` section.
   
3. Commit and push your changes to the git repository configured in the `Clusters` ArgoCD application and wait for it to synchronize.


### Monitoring the installation progress

When ArgoCD `cluster` application synchronizes, two new manifests should appear on the hub cluster generated by the ztp plugin:
- BareMetalHost
- NMStateConfig

As the provisioning is progressing, it might be helpful to monitor following objects:

#### __PreProvisioningImage__
```bash
$ oc get ppimg -A
NAMESPACE   NAME      READY   REASON
cnfdf15     cnfdf15   True    ImageCreated
cnfdf15     cnfdf16   True    ImageCreated

```
#### __BareMetalHost__
```bash
$ oc get bmh -n cnfdf15
NAME      STATE          CONSUMER   ONLINE   ERROR   AGE
cnfdf15   provisioned               true             69m
cnfdf16   provisioning              true             4m50s

```

#### __Agent__

```bash
$ oc get agent -n cnfdf15 --watch
NAME                                   CLUSTER   APPROVED   ROLE     STAGE
671bc05d-5358-8940-ec12-d9ad22804faa   cnfdf15   true       master   Done
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   false               
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   false               
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   false      worker   
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   false      worker   
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   false      worker   
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Starting installation
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Installing
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Writing image to disk
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Writing image to disk
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Writing image to disk
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Writing image to disk
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Writing image to disk
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Writing image to disk
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Writing image to disk
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Writing image to disk
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Writing image to disk
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Writing image to disk
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Writing image to disk
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Writing image to disk
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Writing image to disk
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Writing image to disk
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Writing image to disk
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Writing image to disk
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Writing image to disk
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Writing image to disk
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Waiting for control plane
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Waiting for control plane
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Rebooting
14fd821b-a35d-9cba-7978-00ddf535ff37   cnfdf15   true       worker   Done

```
#### __ManagedClusterInfo__

When the worker node installation completes, its certificates are approved automatically. At this point the worker will appear in ManagedClusterInfo status:

```bash
$ oc get managedclusterinfo/cnfdf15 -n cnfdf15 -o jsonpath='{range .status.nodeList[*]}{.name}{"\t"}{.conditions}{"\t"}{.labels}{"\n"}{end}'
cnfdf15	[{"status":"True","type":"Ready"}]	{"node-role.kubernetes.io/master":"","node-role.kubernetes.io/worker":""}
cnfdf16	[{"status":"True","type":"Ready"}]	{"node-role.kubernetes.io/worker":""}

```
