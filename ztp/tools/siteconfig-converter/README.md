# SiteConfig to ClusterInstance Converter

A command-line tool that converts OpenShift SiteConfig Custom Resources (CRs) to ClusterInstance CRs. The tool provides automated conversion with warnings to the user in limited cases where manual action is required.

### Build from Source

```bash
git clone <repository-url>
cd ztp/tools/siteconfig-converter
make build
```

## Usage

### Basic Usage

```bash
./siteconfig-converter <siteconfig.yaml>
```

### Full Command Syntax

```bash
./siteconfig-converter [-d output_dir] [-t cluster_namespace/name,...] [-n node_namespace/name,...] [-m configmap1,configmap2,...] [-s AgentClusterInstall,ClusterDeployment,...] [-w] [-c] <siteconfig.yaml>
```

### Command-Line Options

| Flag/Argument | Description | Default |
|---------------|-------------|---------|
| `<siteconfig.yaml>` | Path to the SiteConfig YAML file (required positional argument) | - |
| `-d` | Output directory for converted ClusterInstance files | `.` (current directory) |
| `-t` | Comma-separated list of template references for Clusters (format: namespace/name,namespace/name,...) | `open-cluster-management/ai-cluster-templates-v1` |
| `-n` | Comma-separated list of template references for Nodes (format: namespace/name,namespace/name,...) | `open-cluster-management/ai-node-templates-v1` |
| `-m` | Comma-separated list of ConfigMap names for extra manifests references | - |
| `-s` | Comma-separated list of manifest names to suppress at cluster level | - |
| `-w` | Write conversion warnings as comments to the head of converted YAML files | `false` |
| `-c` | Copy comments from the original SiteConfig to the converted ClusterInstance files | `false` |
| `--extraManifestConfigMapName` | Name for the extra manifest ConfigMap | `extra-manifests-cm` |
| `--extraManifestConfigMapNamespace` | Namespace for the extra manifest ConfigMap | Cluster name from SiteConfig |
| `--manifestsDir` | Directory containing extra manifest files | `extra-manifests` |

## Examples

```bash
./siteconfig-converter sno-siteconfig.yaml

./siteconfig-converter -d ./output sno-siteconfig.yaml

# With warnings written to YAML files
./siteconfig-converter -d ./output -w sno-siteconfig.yaml
```

### Custom Templates

```bash
# Use custom cluster and node templates
./siteconfig-converter \
  -d ./output \
  -t my-namespace/custom-cluster-template \
  -n my-namespace/custom-node-template \
  siteconfig.yaml
```

### Multiple Templates

```bash
# Use multiple cluster and node templates (comma-separated)
./siteconfig-converter \
  -d ./output \
  -t cluster-ns1/cluster-template1,cluster-ns2/cluster-template2,cluster-ns3/cluster-template3 \
  -n node-ns1/node-template1,node-ns2/node-template2 \
  siteconfig.yaml

# Mix single and multiple templates
./siteconfig-converter \
  -d ./output \
  -t my-namespace/cluster-template \
  -n node-ns1/node-template1,node-ns2/node-template2,node-ns3/node-template3 \
  siteconfig.yaml
```

#### Creating custom templates 

Refer to siteconfig [docs](https://docs.redhat.com/en/documentation/red_hat_advanced_cluster_management_for_kubernetes/2.13/html/multicluster_engine_operator_with_red_hat_advanced_cluster_management/siteconfig-intro#create-custom-templates) on how to create custom templates. 

#### Creating HostFirmWare custom template instead of biosConfigRef

Create a configmap containing your BIOS config: 

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: custom-host-firmware
  namespace: my-namespace
data:
  HostFirmwareSettings: |-
    apiVersion: metal3.io/v1alpha1
    kind: HostFirmwareSettings
    metadata:
      name: "{{ .SpecialVars.CurrentNode.HostName }}"
      namespace: "{{ .Spec.ClusterName }}"
    spec:
      ### BIOS config content
      # settings:
      #   BootMode: Uefi
```

Generate ClusterInstance by referencing the configmap

```bash
./siteconfig-converter \
  -d ./output \
  -n my-namespace/custom-host-firmware,open-cluster-management/ai-node-templates-v1 \
  siteconfig.yaml
```

### Extra Manifests References

```bash
# Add extra manifests references from ConfigMaps
./siteconfig-converter \
  -m extra-manifests-cm1,extra-manifests-cm2,my-custom-manifests \
  siteconfig.yaml

# Combine with other options
./siteconfig-converter \
  -d ./output \
  -t my-namespace/custom-cluster-template \
  -n my-namespace/custom-node-template \
  -m extra-manifests-cm1,extra-manifests-cm2 \
  siteconfig.yaml

# Combine multiple templates with extra manifests
./siteconfig-converter \
  -d ./output \
  -t cluster-ns1/cluster-template1,cluster-ns2/cluster-template2 \
  -n node-ns1/node-template1,node-ns2/node-template2 \
  -m extra-manifests-cm1,extra-manifests-cm2 \
  siteconfig.yaml
```

#### Generating extraManifest configmap

The `siteconfig-converter` tool automatically generates ConfigMap kustomization file by default.

```bash
# Generate ConfigMap kustomization files automatically
./siteconfig-converter -d ./output siteconfig.yaml
```

This will:
1. Generate extraManifests using the `siteconfig-generator` binary
2. Create a `kustomization.yaml` file

`kustomization.yaml` will add all the `*.yaml` files in the output directory to `resources` field. You can copy other manifests such as `namespace` and `secrets` to output directory before converting, so that they will be included in the `kustmization.yaml` automatically.

The extraManifest configmap will be added by default to `extraManifestRefs` field of `ClusterInstance`.

Example:

```
$ tree .
├── cnfdf28.yaml
├── extra-manifests
│   ├── 98-var-lib-containers-partitioned.yaml
│   └── set-core-user-password.yaml
├── kustomization.yaml
├── ns.yaml
├── secret.yaml

$ siteconfig-converter -d output cnfdf28.yaml
...

$ tree output
output
├── cnfdf28.yaml
├── extra-manifests
│   ├── cnfdf28_machineconfig_98-var-lib-containers-partitioned.yaml
│   └── cnfdf28_machineconfig_99-set-core-user-password.yaml
└── kustomization.yaml
```

**Requirements:**
- `siteconfig-generator` binary must be available in PATH (available in `ztp-site-generator` container)

For the full directory structure refer to siteconfig [docs](https://github.com/stolostron/siteconfig/blob/main/docs/argocd.md#generate-extra-manifests-configmap-using-kustomize).


### Suppressed Manifests

```bash
# Add cluster-level suppressed manifests
./siteconfig-converter \
  -s AgentClusterInstall,ClusterDeployment \
  siteconfig.yaml

# Combine with other options
./siteconfig-converter \
  -d ./output \
  -m extra-manifests-cm1,extra-manifests-cm2 \
  -s AgentClusterInstall,BareMetalHost \
  siteconfig.yaml

# Combine multiple templates with suppressed manifests
./siteconfig-converter \
  -d ./output \
  -t cluster-ns1/cluster-template1,cluster-ns2/cluster-template2 \
  -n node-ns1/node-template1,node-ns2/node-template2 \
  -s AgentClusterInstall,BareMetalHost \
  siteconfig.yaml
```

This tool automatically translates `crSuppressions` from SiteConfig to `suppressedManifests` in ClusterInstance. However when migrating a live cluster, you can suppress `AgentClusterInstall` to avoid its mutation errors. However, make sure to remove the `AgentClusterInstall` suppression when you re-install the cluster.

### Conversion Warnings

By default, conversion warnings are printed to the console. Use the `-w` flag to write warnings as comments to the converted YAML files instead.

```bash
# Default behavior - warnings printed to console
./siteconfig-converter -d ./output siteconfig.yaml

# Write warnings as comments to YAML files
./siteconfig-converter -d ./output -w siteconfig.yaml

# Combine with other options
./siteconfig-converter \
  -d ./output \
  -t cluster-ns1/cluster-template1,cluster-ns2/cluster-template2 \
  -n node-ns1/node-template1,node-ns2/node-template2 \
  -m extra-manifests-cm1,extra-manifests-cm2 \
  -s AgentClusterInstall,BareMetalHost \
  -w \
  siteconfig.yaml
```

When using the `-w` flag, warnings will appear as comments at the head of each converted YAML file:

```yaml
---
# Conversion Warnings:
# - extraManifests field is not supported in ClusterInstance and will be ignored...
# - crTemplates field at cluster level is not supported in ClusterInstance...
#
apiVersion: siteconfig.open-cluster-management.io/v1alpha1
kind: ClusterInstance
metadata:
  name: cluster-name
  namespace: cluster-namespace
spec:
  # ... cluster configuration ...
```

### Copying Comments

Use the `-c` flag to copy comments from the original SiteConfig to the converted ClusterInstance YAML files:

```bash
# Copy comments from SiteConfig to ClusterInstance
./siteconfig-converter -c siteconfig.yaml

# Combine with other options
./siteconfig-converter -d ./output -c -w siteconfig.yaml
```

When using the `-c` flag, comments from the original SiteConfig will be preserved in the ClusterInstance:

#### Field-specific Comments
Comments that are associated with specific fields will be placed near those fields:

```yaml
# Original SiteConfig
spec:
  # Base domain for the cluster
  baseDomain: example.com
  # Pull secret reference for container images
  pullSecretRef:
    name: pull-secret
  clusters:
  - # Cluster configuration
    clusterName: test-cluster
    # Network type for the cluster
    networkType: OVNKubernetes
```

Becomes:

```yaml
# ClusterInstance with copied comments
spec:
  # Base domain for the cluster
  baseDomain: example.com
  # Pull secret reference for container images
  pullSecretRef:
    name: pull-secret
  # Cluster configuration  
  clusterName: test-cluster
  # Network type for the cluster
  networkType: OVNKubernetes
```

#### Header Comments
Comments that are not associated with specific fields (such as top-level comments, metadata comments, or comments on unsupported fields) will be collected and placed in a header section:

```yaml
# Original SiteConfig
# This is a comprehensive SiteConfig example
# Author: Platform Team
apiVersion: ran.openshift.io/v1
kind: SiteConfig
metadata:
  # This is the metadata section
  name: example-cluster
  # Additional labels
  labels:
    environment: production
spec:
  # Base domain for the cluster
  baseDomain: example.com
```

Becomes:

```yaml
---
# Comments from original SiteConfig:
# This is a comprehensive SiteConfig example
# Author: Platform Team
# This is the metadata section
# Additional labels
#
apiVersion: siteconfig.open-cluster-management.io/v1alpha1
kind: ClusterInstance
metadata:
  name: example-cluster
  namespace: example-cluster
spec:
  # Base domain for the cluster
  baseDomain: example.com
```

## Development

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage
```

### Clean Up

```bash
# Clean build artifacts
make clean
```
