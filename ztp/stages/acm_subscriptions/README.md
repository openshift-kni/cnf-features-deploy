# ACM subscription generation

In the RAN ZTP process CR manifests are applied to a cluster by
ACM. These manifests are maintained in a customer's GIT repository to
which ACM subscribes. Changes in the GIT repository are noted by ACM
and the updated manifests are reconciled to the cluster.

This directory contains an Ansible playbook which will generate and
apply to ACM the necessary CR manifests for ACM to subscribe to the
customer's GIT repository.

## Customizing the manifests

The playbook requires an inventory file which contains several
required elements (ex the kubeconfig file for ACM) and optional
configuration items. The elements are documented in
inventory/hosts.sample

## Running the playbook

After updating your inventory file run the playbook:
    ansible-playbook -i <your_inventory_file> playbook.yaml

## License

These playbooks are licensed under the Apache Public License 2.0. The
source code for this program is [located on
github](https://github.com/openshift/origin).
