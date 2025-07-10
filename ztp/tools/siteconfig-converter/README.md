# SiteConfig to ClusterInstance Converter

A command-line tool that converts OpenShift SiteConfig Custom Resources (CRs) to ClusterInstance

### Build from Source

```bash
git clone <repository-url>
cd ztp/tools/siteconfig-converter
make build
```

## Usage

### Basic Usage

```bash
./siteconfig-converter -f <siteconfig.yaml>
```

### Full Command Syntax

```bash
./siteconfig-converter -f <siteconfig.yaml> [-d output_dir] [-t cluster_namespace/name] [-n node_namespace/name] [-m configmap1,configmap2,...] [-s AgentClusterInstall,ClusterDeployment,...]
```

### Command-Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-f` | Path to the SiteConfig YAML file (required) | - |
| `-d` | Output directory for converted ClusterInstance files | `.` (current directory) |
| `-t` | Template reference for ClusterInstance (format: namespace/name) | `open-cluster-management/ai-cluster-templates-v1` |
| `-n` | Template reference for Nodes (format: namespace/name) | `open-cluster-management/ai-node-templates-v1` |
| `-m` | Comma-separated list of ConfigMap names for extra manifests references | - |
| `-s` | Comma-separated list of manifest names to suppress at cluster level | - |

## Examples

### Single Node OpenShift (SNO) Conversion

```bash
./siteconfig-converter -f sno-siteconfig.yaml

./siteconfig-converter -f sno-siteconfig.yaml -d ./output
```

### Custom Templates

```bash
# Use custom cluster and node templates
./siteconfig-converter -f siteconfig.yaml \
  -d ./output \
  -t my-namespace/custom-cluster-template \
  -n my-namespace/custom-node-template
```

### Extra Manifests References

```bash
# Add extra manifests references from ConfigMaps
./siteconfig-converter -f siteconfig.yaml \
  -m extra-manifests-cm1,extra-manifests-cm2,my-custom-manifests

# Combine with other options
./siteconfig-converter -f siteconfig.yaml \
  -d ./output \
  -t my-namespace/custom-cluster-template \
  -n my-namespace/custom-node-template \
  -m extra-manifests-cm1,extra-manifests-cm2
```

### Suppressed Manifests

```bash
# Add cluster-level suppressed manifests
./siteconfig-converter -f siteconfig.yaml \
  -s AgentClusterInstall,ClusterDeployment

# Combine with other options
./siteconfig-converter -f siteconfig.yaml \
  -d ./output \
  -m extra-manifests-cm1,extra-manifests-cm2 \
  -s AgentClusterInstall,BareMetalHost
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
