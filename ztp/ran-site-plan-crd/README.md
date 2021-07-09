# RAN Site Plan Custom Resources definition (CRD)

This directory contain the CRDs that define the required configuration info for RAN site plan

## SiteConfig

Site Config CR has the required configuration to create an Openshift cluster. It has to be used with the policy generator in order to creates the required CRs.

## Note:
  - Secrets need to be created in advance under a namespace same as the cluster name.
    ex:
      ```
      apiVersion: v1
      data:
        password: bmcPasswdBase64
        username: bmcUserBase64
      kind: Secret
      metadata:
        name: du-sno-ex-bmc-secret
        namespace: du-sno-ex
      type: Opaque
      ---
      apiVersion: v1
      kind: Secret
      metadata:
        name: du-sno-ex-pull-secret
        namespace: du-sno-ex
      data:
        .dockerconfigjson: "pullSecretBase64"
      type: kubernetes.io/dockerconfigjson
      ```
  - ClusterImageSet CR need to be created in advance as it is a cluster scope CR can be used across many SiteConfigs
    ex:
      ```
      apiVersion: hive.openshift.io/v1
      kind: ClusterImageSet
      metadata:
        name: openshift-v4.8.0
      spec:
        releaseImage: quay.io/openshift-release-dev/ocp-release:4.8.0-rc.3-x86_64
      ```