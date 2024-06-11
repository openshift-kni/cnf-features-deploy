# Ability to include or exclude CRs at install time
Assisted Installer allows CRs to be applied to SNOs at install time. The applied CRs may include Machine Configs from RAN Far Edge (e.g to enable Workload Partitioning) or CRs defined by the users themselves. More details [here](https://github.com/openshift/assisted-service/blob/c183b5182bfed15e42745e9f7fd3bd4f21184bde/docs/hive-integration/README.md#creating-additional-manifests).

With this feature, via SiteConfig, users can now have control over this process and can perform actions such as removing all or some of the CRs provided at `extraManifests.searchPaths`

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

We continue to support `extraManifestPath` which only accepts a user provided GIT repository path where custom extra manifests are residing. But we strongly recommend the user to adopt the new variable `extraManifests.searchPaths` which:
  * allows to list multiple directory paths on the same GIT repository.  
  * allows same named CR files in different directories and latter directory/filename takes precedence



Variable scenarios are briefly explained in the below table on filtering.

* Assumption: customers siteconfig consists 2 extra manifests paths which are in the user Git repository

  ```yaml
    - cluster:
        extraManifests:
          searchPaths:
            - sno-extra-manifest/
            - custom-manifests/
          filter:			
    ```

  
* Those path conatins below files:

  ```bash
  extra manifests in the sno-extra-manifest

  - 03-sctp-machine-config-worker.yaml
  - B.yaml
  - C.yaml 
  ```

  ```bash
  extra manifests in the custom-manifests

  - C.yaml
  - D.yaml
  - E.yaml 
  ```

<table>
<tr>
  <th>
  Scenario
  </th>
  <th>
  Output CR list
  </th>
</tr>

<tr>
  <td>
  <pre>
  yaml
  - cluster:
      extraManifests:
        filter:
          exclude:
            - 03-sctp-machine-config-worker.yaml
  </pre>
  </td>

  <td>
  remove sctp (worker only) and keep everything else
  </td>
</tr>


<tr>
  <td>
  <pre>
  yaml
  - cluster:
      extraManifests:
        searchPaths:
          - sno-extra-manifest/
          - custom-manifests/	
        filter:
          inclusionDefault: exclude
  </pre>
  </td>
  <td>
  remove all CRs from the install time included in the searchPaths
  </td>
</tr>

<tr>
  <td>
  <pre>
  yaml
  [
    extraManifests:
    searchPaths:
      - sno-extra-manifest/
      - custom-manifests/	
    filter:
      inclusionDefault: exclude
      include:
        - C.yaml
        - D.yaml                  
  ]
  </pre>
  </td>

  <td>
  included files: [C.yaml D.yaml] <--C.yaml picked from <b>custom-manifests</b> path
  </td>
</tr>


<tr>
  <td>
  <pre>
  yaml
  [
  extraManifests:
    searchPaths:
      - sno-extra-manifest/
      - custom-manifests/
    filter:
      inclusionDefault: include
      exclude:
        - C.yaml
        - D.yaml                                 
  ]
  </pre>
  </td>

  <td>
  included files: [03-sctp-machine-config-worker.yaml B.yaml E.yaml]
  </td>
</tr>

</table>
         