**Update RPMs dependencies**

In this project (cnf-tests) we are following the hermetic builds flow to install RPM dependencies according to the Product Security requirments. For more info on why this is needed please follow the info in the following link:
https://konflux.pages.redhat.com/docs/users/faq/hermetic.html.


To delete no longer needed dependencies you need to:
1. remove the binary name from `rpms.in.yaml` and from `Dockerfile`.
2. remove the matching section for this binary from `rpms.lock.yaml`.
3. check if the RPM repo from which this binary is used to install other binaries via searching for same instances in `rpms.lock.yaml`. In case no instances are found, remove the repo section from the relevant `.repo` file.
4. submit the updated files

Or regenerate the file following the procedure in https://konflux.pages.redhat.com/docs/users/how-tos/configuring/activation-keys-subscription.html#_configuring_an_rpm_lockfile_for_hermetic_builds.

To add new dependencies or update binaries versions you need to follow the steps in the docs:
https://konflux.pages.redhat.com/docs/users/how-tos/configuring/activation-keys-subscription.html#_configuring_an_rpm_lockfile_for_hermetic_builds.

As part of the update, make sure that packages are updated in both `rpms.in.yaml` and `Dockerfile` files in order for the installation to be completely network isolated (hermetic).
It is enough that the Dockerfile that is used to generate the lockfile contain the final base image and the command that installs the packages. For example:

```azure
FROM registry.redhat.io/ubi9/ubi-minimal:9.4
RUN microdnf install -y lksctp-tools iproute \
      ethtool iputils procps-ng numactl-libs iptables \
      kmod realtime-tests linuxptp iperf3 nc \
      python3
```

**EUS RPM support**
TBD

**Important**: Please make sure that the repos that you used to pull the RPMs from are found under the activation key that is associated to the konflux public instance by:
Steps on how to confirm this will be detailed later once we have a team activation key. 

**RPM automatic updates**
Konflux uses a mechanism to automatically file PRs to update RPM versions and generate the updated lockfile, and is called Mintmaker. However, this is supported only for repos that have the input file saved in the root of the project repository, which is not an ideal place for cnf-features-deploy considering it produces multiple images (cnf-tests, ztp,..).
The support for multiple data sources (input files) is in progress at https://issues.redhat.com/browse/CWFHEALTH-3922.