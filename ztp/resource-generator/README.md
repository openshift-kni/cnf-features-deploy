## Building the container
The ztp-site-generator container image contains the kustomize plugin binaries SiteConfig and PolicyGenTemplate under /kustomize/plugin directory.
AND contains the mandatory patch files to configure the hub cluster for ztp deployment. The files exist under following directories :
  - /home/ztp/source-crs: contain the source CRs files that SiteConfig & PolicyGenTemplate use to generate the custom resources.
  - /home/ztp/argocd: check the argocd readme for more info on how to configure ArgoCD in the hub cluster.
Run ``` $make build ``` to build ztp-site-generator container image.


## Push the container images to registry
Run ``` $make push ``` in order to publish the image to the registry.

## Test
To export the ``` ztp/ ```  directory from the ztp-site-generator image run the following commands

```
    $ mkdir -p ./out
    $ podman create -ti --name ztp-site-gen ztp-site-generator:latest bash
    $ podman cp ztp-site-gen:/home/ztp ./out
    $ podman rm -f ztp-site-gen
```
Check the created out/ directory you should find source-crs and argocd directories.

You can run ``` $make export ``` in order to copy the binaries and resources from ztp-site-generator container image to ``` out/ ``` directory