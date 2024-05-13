## Configuring IPsec encryption for external traffic

This folder contains the files to configure IPsec encryption for external traffic

* Enable IPsec encryption yaml
* IPsec endpoint configuration with NMState yml
* build.sh script that creates a MachineConfig to import external certs into IPsec NSS and apply the IPsec configuration

Also available in OpenShift docs: [IPsec encryption for external traffic](https://docs.openshift.com/container-platform/4.15/networking/ovn_kubernetes_network_provider/configuring-ipsec-ovn.html#nw-ovn-ipsec-external_configuring-ipsec-ovn)

### Prerequisites

* `butane` utility installed, minimal version 0.20.0.

* IPsec endpoint certificate in PKCS#12 format and a CA cert in PEM format.

### Import external certs and configure IPsec

Provide the following certificate files:

- `left_server.p12`: The certificate bundle for the IPsec endpoints
- `ca.pem`: The certificate authority that you signed your certificates with

Configure the following values in `ipsec-endpoint-config.yml`:

- `$clusterNode` - the IP address or DNS hostname of the cluster node for the IPsec tunnel on the cluster side
- `$externalHost` - the external host IP address or DNS hostname
- `$externalAddress` - the IP address or subnet of the external host

For example:

```
interfaces:
- name: hosta_conn
  type: ipsec
  libreswan:
    left: 10.1.2.3
    leftid: '%fromcert'
    leftmodecfgclient: false
    leftcert: left_server
    leftrsasigkey: '%cert'
    right: 10.1.3.4
    rightid: '%fromcert'
    rightrsasigkey: '%cert'
    rightsubnet: 172.1.2.0/24
    ikev2: insist
    type: tunnel
```

Run the build script that creates a MachineConfig to import the external certs and apply the IPsec configuration:

`./build.sh`

### Deploying IPsec encryption

Include `enable-ipsec.yaml` and `99-ipsec-master-endpoint-config.yaml` files in one of the additional install-time manifests directories defined in the `extraManifests.searchPaths` field in the SiteConfig file.

More info about the `extraManifests.searchPaths` mechanism can be found in the [GitOps ZTP README](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/gitops-subscriptions/argocd/README.md)
