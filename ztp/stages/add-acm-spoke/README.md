This playbook is to allow spoke-clusters to join the ACM hub.

- Requirements:

    - We assume that there is already an ACM hub up and running. If not you can use the ../acm-hub/playbook.yaml to create an ACM hub.
        - Note: The ACM hub needs an independent cluster to run. Cannot use spoke-clusters to run ACM.
        - Note: The ACM must have this subscription_permission manifest https://github.com/redhat-ztp/ztp-acm-manifests/blob/main/hub/04_add_subscription_permission.yaml applied to give proper permissions, otherwise the subscriptions are not created correctly.

- Steps:

    1- git clone the ztp-cluster-deploy repo

    2- go to acm-spoke dir

    3- make a copy inventory/hosts.sample file and rename it as hosts under same directory.

    4- modify the [all:vars] section at inventory/hosts file to match your setup.

    5- Start the deployment.

        $ sudo ansible-playbook playbook.yml -vvv -i inventory/hosts
