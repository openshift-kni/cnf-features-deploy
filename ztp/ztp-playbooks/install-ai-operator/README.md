# Assisted Installer operator (ipv6+disconnected) #
This is a set of playbooks to automate the deployment of Assisted Installer Operator
in a disconnected way, assuming ipv6 networking.

## Requirements ##

1. Podman.
2. Ansible.
3. Ansible modules `containers.podman` and `community.general`. You can install all the requirements with: 

  ```bash
  ansible-galaxy collection install -r requirements.yaml
  ```
        
## Steps ##

1. git clone the `cnf-features-deploy` repo
2. go to `ztp/install-ai-operator` dir
3. You need make a copy `inventory/hosts.sample` file and name it `hosts` under same directory.
4. Modify the [all:vars] section at `inventory/hosts` file based on your env.
5. If you want that the playbook creates the initial provisioner cluster for you, please provide
the provisioner_cluster details under inventory/hosts. If not, please just provide the kubeconfig
path of an already provisioned cluster.
6. Start the deployment with `prepare-environment` tag:

      ```console
      $ sudo ansible-playbook playbook.yml -vvv -i inventory/hosts --tags=prepare-environment
      ```

7. Create the OpenShift disconnected mirror if needed. You may use another pre-created mirror, by
providing the provisioner_cluster_registry var into inventory/hosts. If not, please run this playbook,
and it will create a mirror on a virtual machine connected to the provisioner cluster that was created
before:

      ```console
      $ sudo ansible-playbook playbook.yml -vvv -i inventory/hosts --tags=offline-mirror-olm
      ```

8. Mirror Assisted Installer images. It will just use the default mirror settings, or it can also mirror
into an existing one, by providing the `provisioner_cluster_registry` var. It will also create a local
http server, where it will copy the RHCOS images. By default it will use the same VM used for the
disconnected registry, but you can provide your own using the `disconnected_http_server` var:

      ```console
      $ sudo ansible-playbook playbook.yml -vvv -i inventory/hosts --tags=mirror-ai-images
      ```
    
9. Finally install the Assisted Installer operator. It will be installing in the default provisioner cluster,
but you can provide your own by setting de `provisioner_cluster_kubeconfig` var in inventory:

      ```console
      $ sudo ansible-playbook playbook.yml -vvv -i inventory/hosts --tags=install-assisted-installer
      ```

10. Once deployed, you could create clusters using CRDs. This is going to be covered on a different section.
