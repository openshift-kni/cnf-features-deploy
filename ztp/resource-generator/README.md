## Building the container
The ztp-site-generator container image contains the kustomize plugin binaries SiteConfig and PolicyGenTemplate under /kustomize/plugin directory.
AND contains the mandatory patch files to configure the hub cluster for ztp deployment. The files exist under following directories :
  - /home/ztp/source-crs: contain the source CRs files that SiteConfig and PolicyGenTemplate use to generate the custom resources.
  - /home/ztp/argocd: check the argocd readme for more info on how to configure ArgoCD in the hub cluster.
  - Run ``` $make build ``` to build ztp-site-generator container image.

## Export
Run ``` $ make export ```  to export the ztp-site-generator container image directories.

```
$ tree out/ -L 2
out/
├── exportkustomize.sh
├── kustomize
│   └── plugin
└── ztp
    ├── argocd
    ├── extra-manifest
    └── source-crs
```

## Automatic upstream container builds
The Red Hat Prow infractructure automatically pushes the head of this
master branch to quay.io/openshift-kni/ztp-site-generator:latest
