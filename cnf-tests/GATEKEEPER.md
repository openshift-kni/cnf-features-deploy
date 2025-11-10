# Gatekeeper testing

This document describes how to test set up and test Gatekeeper manually.
It describes how to install Gatekeeper using the [gatekeeper-operator](https://github.com/openshift/gatekeeper-operator), setting up some basic mutations and how to validate the mutations are applied correctly.

## Gatekeeper setup

The Gatekeeper operator can be installed by running the following command in the root of the [cnf-features-deploy](https://github.com/openshift-kni/cnf-features-deploy) repo:

```bash
FEATURES=gatekeeper make feature-deploy
```

Once the operator is running, set up a Gatekeeper instance by running the following command:
 
```shell
cat << EOF | oc create -f -
apiVersion: operator.gatekeeper.sh/v1alpha1
kind: Gatekeeper
metadata:
  name: gatekeeper
spec:
  audit:
    replicas: 1
  mutatingWebhook: "Enabled"
  webhook:
    failurePolicy: "Ignore"
EOF
```

Check if the gatekeeper deployments are running by executing the following command:

```shell
oc get deployment -n gatekeeper-system
```

The output should show the following deployments to be ready:
- gatekeeper-audit
- gatekeeper-controller-manager

for example:
```shell
NAME                                                     READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/gatekeeper-audit                         1/1     1            1           12s
deployment.apps/gatekeeper-controller-manager            1/1     1            1           12s
```

## Testing Gatekeeper constraints

Sample constraint templates and constraints can be found in ` feature-configs/demo/gatekeeper`.

To apply these please run:

```shell
oc create -f feature-configs/demo/gatekeeper/00_ConstraintTemplates.yaml
oc create -f feature-configs/demo/gatekeeper/10_Constraints.yaml 
```

"NOTE:" please wait a few seconds between the commands to allow gatekeeper to process them.

With the constraints in place, create some pods to validate that the constraints are applied properly.
To restrict the effect of the validation on the cluster, the constraints are only valid for namespaces having a `admission.gatekeeper.sh/tolerations: "enforce"` label.

Create such a namespace by executing:

```shell
oc create -f feature-configs/demo/gatekeeper/20_DemoNamespace.yaml
```

#### Create a basic pod passing validation

Create the following pod:

```shell
cat << EOF | oc create -f -
apiVersion: v1
kind: Pod
metadata:
  name: pod1
spec:
  containers:
  - name: main
    image: centos
    command: ["/bin/bash", "-c", "sleep INF"]
EOF    
```

The pod should be created successfully.

#### Create a pod failing validation

Create the following pods:

##### Pod with a specific toleration

```shell
cat << EOF | oc create -f -
apiVersion: v1
kind: Pod
metadata:
  name: pod2
  namespace: gatekeeper-demo
spec:
  tolerations:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
    operator: Exists
  containers:
  - name: main
    image: centos
    command: ["/bin/bash", "-c", "sleep INF"]
EOF
```

Gatekeeper will reject this pod with a message similar to the one below:

```
Error from server ([deny-master-no-schedule-toleration] Toleration is not allowed for taint {"effect": "NoSchedule", "key": "node-role.kubernetes.io/master", "value": "true"}): error when creating "STDIN": admission webhook "validation.gatekeeper.sh" denied the request: [deny-master-no-schedule-toleration] Toleration is not allowed for taint {"effect": "NoSchedule", "key": "node-role.kubernetes.io/master", "value": "true"}
```

This pod was rejected by the deny-master-no-schedule-toleration constraint of type k8srestrictspecifictoleration.constraints.gatekeeper.sh

##### Pod with a global taint

```shell
cat << EOF | oc create -f -
apiVersion: v1
kind: Pod
metadata:
  name: pod4
  namespace: gatekeeper-demo
spec:
  tolerations:
  - operator: "Exists"
  containers:
  - name: podexample
    image: centos
    command: ["/bin/bash", "-c", "sleep INF"]
EOF
```

Gatekeeper will reject this pod with a message similar to the one below:

```
The Pod "pod4" is invalid: spec.tolerations: Forbidden: existing toleration can not be modified except its tolerationSeconds
```

This pod was rejected by the deny-global-tolerations constraint of type k8srestrictglobaltoleration.constraints.gatekeeper.sh