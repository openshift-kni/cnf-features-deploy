## Building the container
The ztp-site-generator container image contains the kustomize plugin binaries SiteConfig and PolicyGenerator under /kustomize/plugin directory.
AND contains the mandatory patch files to configure the hub cluster for ztp deployment. The files exist under following directories :
  - /home/ztp/acmpolicygenerator: PolicyGenerator CRs for all supported DU configurations (recommended).
  - /home/ztp/policygentemplates: deprecated PolicyGenTemplate CRs (use PolicyGenerator instead).
  - /home/ztp/template-values: ConfigMaps for hub-side templating (hardware-types, zones, site-data).
  - /home/ztp/source-crs: contain the source CRs files that PolicyGenerator uses to generate the custom resources.
  - /home/ztp/argocd: check the argocd readme for more info on how to configure ArgoCD in the hub cluster.
  - Run ``` $make build ``` to build ztp-site-generator container image.

## Export
Run ``` $ make export ```  to export the ztp-site-generator container image directories.

```
$ tree out/ -L 2
out/
├── exportkustomize.sh
├── kustomize
│   └── plugin
└── ztp
    ├── acmpolicygenerator
    ├── policygentemplates
    ├── template-values
    ├── argocd
    ├── extra-manifest
    └── source-crs
```

## Automatic upstream container builds
The Red Hat Prow infractructure automatically pushes the head of this
master branch to quay.io/openshift-kni/ztp-site-generator:latest

## Custom builds
The argocd deployment files refer to the upstream container images by
default. But downstream builds (or other special-purpose builds) need
to override that internal reference.  Setting the IMAGE_REF build
argument will patch any internal references to that argument's value
verbatim.

Example:
```
make build IMAGE_REF=quay.io/personal/ztp-site-generator:latest
```
