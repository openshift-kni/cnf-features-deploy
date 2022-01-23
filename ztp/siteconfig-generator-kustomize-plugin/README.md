## SiteConfig generator kustomize plugin

The siteConfig generator kustomize plugin consume the [siteconfig-generator](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/siteconfig-generator) lib as a kustomize plugin. Kustomization.yaml is an example how to use the siteconfig plugin as below
```
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

generators:
- testSiteConfig/site1-sno-du.yaml
- testSiteConfig/site2-sno-du.yaml
- testSiteConfig/site2-standard-du.yaml
```


# Build and execute
- Run the following command to build siteconfig-generator binary and create the kustomize plugin directory
```
    $ make build
```
You should see after the build success `kustomize/` directory should be created with following tree
```
├── kustomize
│   └── plugin
│       └── ran.openshift.io
│           └── v1
│               └── siteconfig
│                   ├── extra-manifest
│                   │   ├── 01-container-mount-ns-and-kubelet-conf.yaml
│                   │   ├── 03-sctp-machine-config.yaml
│                   │   ├── 05-chrony-dynamic.yaml
│                   │   ├── disk-encryption.yaml.tmpl
│                   │   └── workload
│                   │       ├── 03-workload-partitioning.yaml
│                   │       ├── crio.conf
│                   │       └── kubelet.conf
│                   └── SiteConfig

```
The extra-manifest directory contain the MCs (Machine Configs) that will be associated with clusters

- Run the following command to execute kustomization.yaml
```
    $ make test
```

- Run the following command to dump the kusomization output to files under `out/` directory
```
    $ make gen-files
```
