#Ability to include or exclude CRs at install time
Assisted Installer allows CRs to be applied to SNOs at install time. The applied CRs may include Machine Configs from RAN Far Edge (e.g to enable Workload Partitioning) or CRs defined by the users themselves. More details [here](https://github.com/openshift/assisted-service/blob/c183b5182bfed15e42745e9f7fd3bd4f21184bde/docs/hive-integration/README.md#creating-additional-manifests).

With this feature, via SiteConfig, users can now have control over this process and can perform actions such as removing all or some of the CRs provided RAN or their own set of CRs.

```yaml
- cluster:
    extraManifests:
      filter:
        inclusionDefault: [include|exclude]
        exclude:
          - CR1
          - CR3
        include:
          - CR1
          - CR3
```
## Use Cases

- Remove sctp (worker only) and keep everything else
```yaml
- cluster:
    extraManifests:
      filter:
        exclude:
          - 03-sctp-machine-config-worker.yaml
```

- Remove everything from install time include CRs provided by RAN and user defined
```yaml
- cluster:
    extraManifests:
      filter:
        inclusionDefault: exclude
```

- Keep only user defined called `myCR.yaml`. This will not include RAN files e.g sctp, generated yaml from .tmpl file, and any other user provided files. 
```yaml
- cluster:
    extraManifestPath: mypath/
    extraManifests:
      filter:
        inclusionDefault: exclude
        include:
          - myCR.yaml
    nodes:
      - hostName: "node1"
        diskPartition:
          - device: /dev/sda
            partitions:
              - mount_point: /var/imageregistry
                size: 102500
                start: 344844
```
