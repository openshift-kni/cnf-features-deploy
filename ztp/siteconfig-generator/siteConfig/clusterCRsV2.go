package siteConfig

// SiteConfigV2 uses clusterCRsV2 to generate the required CRs for cluster provisioning

const siteConfigAPIV2 = siteConfigAPIGroup + "/v2"

const clusterCRsV2 = `
apiVersion: v1
kind: Namespace
metadata:
  name: "{{ .Cluster.ClusterName }}"
  labels:
    name: "{{ .Cluster.ClusterName }}"
  annotations:
    argocd.argoproj.io/sync-wave: "0"
---
apiVersion: extensions.hive.openshift.io/v1beta1
kind: AgentClusterInstall
metadata:
  name: "{{ .Cluster.ClusterName }}"
  namespace: "{{ .Cluster.ClusterName }}"
  annotations:
    agent-install.openshift.io/install-config-overrides: "{{ .Cluster.NetworkType }}"
    argocd.argoproj.io/sync-wave: "1"
spec:
  clusterDeploymentRef:
    name: "{{ .Cluster.ClusterName }}"
  holdInstallation: "{{ .Cluster.HoldInstallation }}"
  imageSetRef:
    name: "{{ .Cluster.ClusterImageSetNameRef }}"
  apiVIP: "{{ .Cluster.ApiVIP }}"
  ingressVIP: "{{ .Cluster.IngressVIP }}"
  apiVIPs: "{{ .Cluster.ApiVIPs }}"
  ingressVIPs: "{{ .Cluster.IngressVIPs }}"
  networking:
    clusterNetwork: "{{ .Cluster.ClusterNetwork }}"
    machineNetwork: "{{ .Cluster.MachineNetwork }}"
    serviceNetwork: "{{ .Cluster.ServiceNetwork }}"
  provisionRequirements:
    controlPlaneAgents: "{{ .Cluster.NumMasters }}"
    workerAgents: "{{ .Cluster.NumWorkers }}"
  proxy: "{{ .Cluster.ProxySettings }}"
  sshPublicKey: "{{ .Site.SshPublicKey }}"
  manifestsConfigMapRefs: 
    "{{ .Cluster.ManifestsConfigMapRefs }}"
---
apiVersion: hive.openshift.io/v1
kind: ClusterDeployment
metadata:
  name: "{{ .Cluster.ClusterName }}"
  namespace: "{{ .Cluster.ClusterName }}"
  annotations:
    argocd.argoproj.io/sync-wave: "1"
spec:
  baseDomain: "{{ .Site.BaseDomain }}"
  clusterInstallRef:
    group: extensions.hive.openshift.io
    kind: AgentClusterInstall
    name: "{{ .Cluster.ClusterName }}"
    version: v1beta1
  clusterName: "{{ .Cluster.ClusterName }}"
  platform:
    agentBareMetal:
      agentSelector:
        matchLabels:
          cluster-name: "{{ .Cluster.ClusterName }}"
  pullSecretRef:
    name: "{{ .Site.PullSecretRef.Name }}"
---
apiVersion: agent-install.openshift.io/v1beta1
kind: NMStateConfig
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "1"
  name: "{{ .Node.HostName }}"
  namespace: "{{ .Cluster.ClusterName }}"
  labels:
    nmstate-label: "{{ .Cluster.ClusterName }}"
spec:
  config: "{{ .Node.NodeNetwork.Config }}"
  interfaces: "{{ .Node.NodeNetwork.Interfaces }}"
---
apiVersion: agent-install.openshift.io/v1beta1
kind: InfraEnv
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "1"
  name: "{{ .Cluster.ClusterName }}"
  namespace: "{{ .Cluster.ClusterName }}"
spec:
  clusterRef:
    name: "{{ .Cluster.ClusterName }}"
    namespace: "{{ .Cluster.ClusterName }}"
  sshAuthorizedKey: "{{ .Site.SshPublicKey }}"
  proxy: "{{ .Cluster.ProxySettings }}"
  pullSecretRef:
    name: "{{ .Site.PullSecretRef.Name }}"
  ignitionConfigOverride: "{{ .Cluster.IgnitionConfigOverride }}"
  nmStateConfigLabelSelector:
    matchLabels:
      nmstate-label: "{{ .Cluster.ClusterName }}"
  additionalNTPSources: "{{ .Cluster.AdditionalNTPSources }}"
---
apiVersion: metal3.io/v1alpha1
kind: BareMetalHost
metadata:
  name: "{{ .Node.HostName }}"
  namespace: "{{ .Cluster.ClusterName }}"
  annotations:
    argocd.argoproj.io/sync-wave: "1"
    inspect.metal3.io: "{{ .Node.IronicInspect }}"
    bmac.agent-install.openshift.io.node-label: "{{ .Node.NodeLabels }}"
    bmac.agent-install.openshift.io/hostname: "{{ .Node.HostName }}"
    bmac.agent-install.openshift.io/installer-args: "{{ .Node.InstallerArgs }}"
    bmac.agent-install.openshift.io/ignition-config-overrides: "{{ .Node.IgnitionConfigOverride }}"
    bmac.agent-install.openshift.io/role: "{{ .Node.Role }}"
  labels:
    infraenvs.agent-install.openshift.io: "{{ .Cluster.ClusterName }}"
spec:
  bootMode: "{{ .Node.BootMode }}"
  bmc:
    address: "{{ .Node.BmcAddress }}"
    disableCertificateVerification: true
    credentialsName: "{{ .Node.BmcCredentialsName.Name }}"
  bootMACAddress: "{{ .Node.BootMACAddress }}"
  automatedCleaningMode: "{{ .Node.AutomatedCleaningMode }}"
  online: true
  rootDeviceHints: "{{ .Node.RootDeviceHints }}"
  userData:  "{{ .Node.UserData }}"
  # TODO: https://github.com/openshift-kni/cnf-features-deploy/issues/619
---
apiVersion: metal3.io/v1alpha1
kind: HostFirmwareSettings
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "1"
  name: "{{ .Node.HostName }}"
  namespace: "{{ .Cluster.ClusterName }}"
spec: "{{ .Node.BiosConfigRef.FilePath }}"
---
# Extra manifest will be added to the data section
kind: ConfigMap
apiVersion: v1
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "1"
  name: "{{ .Cluster.ClusterName }}"
  namespace: "{{ .Cluster.ClusterName }}"
data:
---
kind: ConfigMap
apiVersion: v1
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "2"
  name: "{{ .Cluster.SiteConfigMap.Name }}"
  namespace: "{{ .Cluster.SiteConfigMap.Namespace }}"
data: "{{ .Cluster.SiteConfigMap.Data }}"
---
apiVersion: cluster.open-cluster-management.io/v1
kind: ManagedCluster
metadata:
  name: "{{ .Cluster.ClusterName }}"
  labels: "{{ .Cluster.ClusterLabels }}"
  annotations:
    argocd.argoproj.io/sync-wave: "2"
spec:
  hubAcceptsClient: true
---
apiVersion: agent.open-cluster-management.io/v1
kind: KlusterletAddonConfig
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "2"
  name: "{{ .Cluster.ClusterName }}"
  namespace: "{{ .Cluster.ClusterName }}"
spec:
  clusterName: "{{ .Cluster.ClusterName }}"
  clusterNamespace: "{{ .Cluster.ClusterName }}"
  clusterLabels:
    cloud: auto-detect
    vendor: auto-detect
  applicationManager:
    enabled: false
  certPolicyController:
    enabled: false
  iamPolicyController:
    enabled: false
  policyController:
    enabled: true
  searchCollector:
    enabled: false
`
