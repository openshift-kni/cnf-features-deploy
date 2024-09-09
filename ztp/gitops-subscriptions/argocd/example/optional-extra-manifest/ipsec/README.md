## Configuring IPsec encryption for external traffic

This folder contains the files to configure IPsec encryption for external traffic

Also available in OpenShift docs: [IPsec encryption for external traffic](https://docs.openshift.com/container-platform/4.16/networking/ovn_kubernetes_network_provider/configuring-ipsec-ovn.html#nw-ovn-ipsec-external_configuring-ipsec-ovn)

### Prerequisites

* `butane` utility installed, minimal version 0.20.0.
* Include `enable-ipsec.yaml` file in one of the additional install-time manifests directories defined in the `extraManifests.searchPaths` field in the SiteConfig file.
More info about the `extraManifests.searchPaths` mechanism can be found in the [GitOps ZTP README](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/gitops-subscriptions/argocd/README.md)
* Provide the following certificate files:
  - `left_server.p12`: The certificate bundle for the IPsec endpoints
  - `ca.pem`: The certificate authority that you signed your certificates with

### IPsec configuration with NMState Operator

> The use of NMState Operator is supported for MNO clusters only due to resource consumption, to configure IPsec on SNO and SNO+1 clusters continue with [IPsec configuration with a MachineConfig](#ipsec-configuration-with-a-machineconfig)

#### Prerequisites

* NMState Operator installed on the cluster. The configuration can be found in the `group-du-3node-ranGen.yaml` or `group-du-standard-ranGen.yaml` policy generator template.

#### Import external certs

Use `import-certs.sh` script that creates a MachineConfig to import the external certs.

- If the PKCS#12 certificate is protected with a password set `-W` argument.

Run `./import-certs.sh` to build the MachineConfig.

Include `99-ipsec-master-import-certs.yaml` and `99-ipsec-worker-import-certs.yaml` files in one of the additional install-time manifests directories.

#### Configure IPsec

Create an IPsec configuration with an NMState Operator node network configuration policy.

Configure the following values in `ipsec-config-policy.yaml`:

- `$externalHost` - the external host IP address or DNS hostname
- `$externalAddress` - the IP address or subnet of the external host

Include the config policy file `ipsec-config-policy.yaml` in source-crs directory in gitops and reference the file in one of the PolicyGenerator files.

### IPsec configuration with a MachineConfig

#### Import external certs and configure IPsec

Use `configure-ipsec.sh` script that creates a MachineConfig to import the external certs and apply the IPsec configuration.

- If the PKCS#12 certificate is protected with a password set `-W` argument.

Configure the following values in `ipsec-endpoint-config.yml`:

- `$externalHost` - the external host IP address or DNS hostname
- `$externalAddress` - the IP address or subnet of the external host

Run `./configure-ipsec.sh` to build the MachineConfig 

#### Deploying IPsec encryption

Include `99-ipsec-master-endpoint-config.yaml` and `99-ipsec-worker-endpoint-config.yaml` files in one of the additional install-time manifests directories.
