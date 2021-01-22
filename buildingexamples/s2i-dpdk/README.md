
# Basic S2I builder image for dpdk workloads

This folder contains files for building a s2i dpdk base image.

This folder also contains a simple dpdk workload test inside the `test/test-app` folder.
The test application use `expect` to run `testpmd`.

# How to use

To use this s2i you only need to run the [build-config.yaml](base-image/build-config.yaml).

This will create a new "dpdk" namespace then it will configure a "ImageStream" for the image and start a build. 

After the base dpdk image build is ready you should create a new directory under the [feature-configs](../../feature-configs/) folder
with a `kustomization.yaml` and a `build-config.yaml` patch.

#### Example:

`kustomization.yaml` file:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

bases:
  - ../demo/dpdk
patchesStrategicMerge:
  - build-config.yaml
``` 

`build-config.yaml` file:

```yaml
---
apiVersion: build.openshift.io/v1
kind: BuildConfig
metadata:
  name: s2i-dpdk
  namespace: dpdk
spec:
  strategy:
    sourceStrategy:
      from:
        kind: ImageStreamTag
        name: dpdk-s2i-base:latest
    type: Source
```

Then you can just deploy the new folder using kustomize.

```bash
oc apply -k feature-configs/<new-patch-folder>
```