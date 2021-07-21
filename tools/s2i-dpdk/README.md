
# Basic Source2image builder image for dpdk workloads

The dpdk-base Source2image builder image is a base image for dpdk based applications, making use of the Source2image build tool.

The dpdk-base image is preinstalled with the dpdk library and the Source2image build tool (for more information about the Source2image tool check [here](https://github.com/openshift/source-to-image)). It can be used to create a target image from a user provided application.

# User application

The user can use the dpdk-base image to build his own dpdk based applications.
The dpdk-base image and the user application are both used to build a target application image. Source2image will copy the user application source code into the dpdk-base image, which will then build a target image using dpdk-base image resources and the copied application.

For a user application to be consumed by the dpdk-base Source2image builder image, the application must have two scripts in its root directory:
 - `build.sh` - used for building the application. It will be used by the builder image when the target image is build by the Source2image build tool
 - `run.sh` - used for running the application. It will be used for running the user application on the target image produced by the Source2image build tool

A sample application is shown below.

# Sample appication

A simple dpdk workload test application is located inside the [`test/test-app`](https://github.com/openshift-kni/cnf-features-deploy/tree/master/tools/s2i-dpdk/test/test-app) folder.
The example application is the [test-pmd application](https://doc.dpdk.org/guides/testpmd_app_ug/) provided by dpdk.org.
Note the `build.sh` and `run.sh` files in the application root directory.

# How to use

The user can build a target image containing the user application by means of a BuildConfig. A sample defintion of a BuildConfig can be found here: [build-config.yaml](base-image/build-config.yaml). This will do a Source2image build of the user application and place it on a target image.


This [build-config.yaml](base-image/build-config.yaml) will create a new "dpdk" namespace then it will configure a "ImageStream" for the image and start a build. 

After the base dpdk image build is ready you should create a new directory under the [feature-configs](../../feature-configs/) folder
with a `kustomization.yaml` and a `build-config.yaml` patch.

The build can be inspected using the following commands:

```
oc -n dpdk get bc
oc -n dpdk get build
oc logs -f bc/dpdk-base
```

#### Example:

kustomization.yaml file:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

bases:
  - ../demo/dpdk
patchesStrategicMerge:
  - build-config.yaml
``` 

build-config.yaml file:

```yaml
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