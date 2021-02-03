# SCTP Application building example

This repo showcases how to build a simple C sctp application using [docker build strategies](https://docs.openshift.com/container-platform/4.3/builds/understanding-image-builds.html#builds-strategy-docker-build_understanding-image-builds).
The C client application is just a small sctp client that will send what is passed from the command line, but should be fine for the purpouse of this example.

## Consuming a ubi8 image

Using `ubi8` images (and installing additional dependencies), requires a working subscription.

The process of doing that is described in the [OpenShift docs](https://docs.openshift.com/container-platform/4.3/builds/running-entitled-builds.html).

### Quick guide

#### Start up

Create an ImageStream to reference the universal base image (UBI):

```bash
oc tag --source=docker registry.redhat.io/ubi8/ubi:latest ubi:latest
```

In order to install the required dependencies a working subscription is needed, as described in [this documentation](https://docs.openshift.com/container-platform/4.3/builds/running-entitled-builds.html).

Here below we try to sum up what's needed.

Fetch your subscription manager's subscription's entitlement and create a secret with them:

```bash
oc create secret generic etc-pki-entitlement --from-file entitlement-key.pem --from-file entitlement.pem
```

The public / private entitlement keys can be found under `/etc/pki/entitlement/`

Add your subscription manager's configuration and certificate authority as config maps:

```bash
oc create configmap rhsm-conf --from-file rhsm.conf
oc create configmap rhsm-ca --from-file redhat-uep.pem
```

`redhat-uep.pem` can be found under `/etc/rhsm/ca/redhat-uep.pem`
`rhsm.conf` can be found under `/etc/rhsm/rhsm.conf`

Empty files can be found under [setup_sample](setup_sample).

#### Configuring the build

A sample build configuration leveraging the secrets / config maps just created can be found in [buildconfig](buildconfig/buildconfig.yaml).

Docker build strategies are described [in the official OpenShift doc](https://docs.openshift.com/container-platform/4.3/builds/build-strategies.html#builds-strategy-docker-build_build-strategies).

### Launch the build

An `imagestream` matching the one we are building must be created, like

```bash
oc create imagestream sctp-sample
```

Once the build configuration is applied, we can start a new build:

```bash
oc start-build --build-loglevel 5 sctp-sample-build
```

and track it:

```bash
oc logs -f bc/sctp-sample-build
```

If everything goes fine, the new image will be ready to be used.
