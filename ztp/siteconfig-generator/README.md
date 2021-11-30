## SiteConfig generator

The siteconfig-generator library makes cluster deployment easier by generating the following CRs based on a SiteConfig CR instance;
  - AgentClusterInstall
  - ClusterDeployment
  - NMStateConfig
  - KlusterletAddonConfig
  - ManagedCluster
  - InfraEnv
  - BareMetalHost
  - ConfigMap for extra-manifest configurations

The [SiteConfig](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/ran-crd/site-config-cr-ex.yaml) is a Custom Resource created to facilitate the creation of those CRs and avoid repeating the configuration names.

# Build and execute
- Run the following command to build siteconfig-generator binary
```
    $ make build
```

- Run the following command to execute the unit test
```
    $ make test
```

- Run the following command to execute siteconfig-generator binary with a SiteConfig example
```
    $ ./siteconfig-generator ../siteconfig-generator-kustomize-plugin/testSiteConfig/site1-sno-du.yaml
```
