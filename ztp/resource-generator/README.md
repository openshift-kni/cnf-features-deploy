# CR Generation Container

The infrastructure in this directory will build a container image capable of executing the [PolicyGenerator](../ztp-policy-generator/README.md) tool for generation of CRs from SiteConfig or PolicyGenTemplate objects.

The following diagram shows a high level view of how the container is used in generating CRs:
![ ZTP flow overview](assets/flow.png)

## Building the container
The included Makefile will build both the base and hook container images.

## Updating source CRs
If additional CRs are needed during installation they may be added to the `/usr/src/hook/ztp/source-crs/extra-manifest/` directory. Similarly additional configuration CRs, as referenced from a PolicyGenTemplate, may be added to the `/usr/src/hook/ztp/source-crs/` directory. The container may be rebuilt with these additional files:  

```
FROM localhost/ztp-site-generator:latest

COPY myInstallManifest.yaml /usr/src/hook/ztp/source-crs/extra-manifest/
COPY mySourceCR.yaml /usr/src/hook/ztp/source-crs/
```

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
