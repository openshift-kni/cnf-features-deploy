**Update RPMs dependencies**

In this project (cnf-tests) we are following the hermetic builds flow to install RPM dependencies according to the Product Security requirments. For more info on why this is needed please follow the info in the following link:
https://konflux.pages.redhat.com/docs/users/faq/hermetic.html.


To delete no longer needed dependencies you need to:
1. remove the binary name from `rpms.in.yaml` and from `Dockerfile`.
2. remove the matching section for this binary from `rpms.lock.yaml`.
3. check if the RPM repo from which this binary is used to install other binaries via searching for same instances in `rpms.lock.yaml`. In case no instances are found, remove the repo section from the relevant `.repo` file or update repos section in `rpms.in.yaml`.
4. submit the updated files

Or regenerate the file following the procedure in https://konflux.pages.redhat.com/docs/users/how-tos/configuring/activation-keys-subscription.html#_configuring_an_rpm_lockfile_for_hermetic_builds.

To add new dependencies or update binaries versions you need to follow the steps in the docs:
https://konflux.pages.redhat.com/docs/users/how-tos/configuring/activation-keys-subscription.html#_configuring_an_rpm_lockfile_for_hermetic_builds.

As part of the update, make sure that packages are updated in both `rpms.in.yaml` and `Dockerfile` files in order for the installation to be completely network isolated (hermetic).
It is enough that the Dockerfile that is used to generate the lockfile contain the final base image and the command that installs the packages. For example:

```azure
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.4
RUN microdnf install -y lksctp-tools iproute \
      ethtool iputils procps-ng numactl-libs iptables \
      kmod realtime-tests linuxptp iperf3 nc \
      python3
```

**EUS RPM support**
When an image version is out-of-maintenance (OOM) some versions has what's called extended-update-support (EUS) period where critical security and CVE fixes will still be shipped. The container image that us built on top of an EUS base should also prefetch the dependencies from corresponding EUS RPM repositories.
As any other RPM repo, also EUS repos need to be enabled in the activation key. Once enabled, the lockfile will be generated with additional EUS packages. The version of the base images should anyhow align with those used for OCP for the same branch.   

**Important**: 
* When starting the container in which you will be generating the lockfile in, use a production image in order to get the GA RPM repos and not beta one. So use `registry.access.redhat.com/ubi9/ubi-minimal:9.4` and not `registry-proxy.engineering.redhat.com/rh-osbs/ubi9/ubi-minimal:9.4`.
* Please make sure that the repos that you used to pull the RPMs from are found under the activation key that is associated to the konflux public instance by:
<steps on how to confirm this will be detailed later once we have a team activation key> 

**RPM automatic updates**
Konflux uses a mechanism to automatically file PRs to update RPM versions and generate the updated lockfile, and is called Mintmaker. However, this is supported only for repos that have the input file saved in the root of the project repository, which is not an ideal place for cnf-features-deploy considering it produces multiple images (cnf-tests, ztp,..).
The support for multiple data sources (input files) is in progress at https://issues.redhat.com/browse/CWFHEALTH-3922.

**Manual RPM lock regeneration using Makefile**
To manually regenerate the rpm lock configuration for cnf-tests, use the following Makefile targets from the `cnf-tests/` directory.

1. **Update rpm lock file for runtime:**
   ```bash
   make konflux-update-rpm-lock-runtime
   ```
   This target will:
   - Sync git submodules
   - Copy and process the Dockerfile to the lock-runtime directory
   - Generate RHEL9 locks using the base image specified in the Dockerfile
   - Automatically extract RHEL9 release version from the Dockerfile
   - Update the `.konflux/lock-runtime/rpms.lock.yaml` file
   - Clean up temporary files

**Configuration Options:**
- `RHEL9_RELEASE`: RHEL9 release version (automatically extracted from Dockerfile)
- `RHEL9_ACTIVATION_KEY`: Red Hat activation key for RHEL9 (required if using subscription-based repos)
- `RHEL9_ORG_ID`: Red Hat organization ID for RHEL9 (required if using subscription-based repos)

**Example with custom activation key:**
```bash
make konflux-update-rpm-lock-runtime RHEL9_ACTIVATION_KEY="your-activation-key" RHEL9_ORG_ID="your-org-id"
```
