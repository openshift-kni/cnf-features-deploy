## SiteConfig generator

The siteconfig-generator library makes cluster deployment easier by generating the following CRs based on a SiteConfig CR instance;
  - AgentClusterInstall
  - ClusterDeployment
  - NMStateConfig
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
    $ ./siteconfig-generator -manifestPath ../source-crs/extra-manifest ../siteconfig-generator-kustomize-plugin/testSiteConfig/site2-sno-du.yaml
```
Note: the manifestPath option is to set the predefined extra-manifest path exist under ../source-crs/extra-manifest

- For using siteconfig-generator library as kustomize plugin check the [siteconfig-generator-kustomize-plugin](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/siteconfig-generator-kustomize-plugin/README.md)
