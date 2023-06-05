# Logging and log forwarding configuration
This example shows how to configure cluster logging and cluster log forwarding to send logs to a Kafka broker.
## Logging concepts
Cluster logging is handled by cluster logging operator. The operator implements two custom resources:
- `ClusterLogging` - Creates `fluentd` daemonset (fluentd is the collector)
- `ClusterLogForwarding`: 
    - Configures the `fluentd` instances to collect logs of specific types from the specified `inputs`
    - Defines the log remote collectors as `outputs`
    - Defines `pipelines` that connect sets of `inputs` to sets of `outputs`
    
## Operator installation
The cluster logging operator can be installed from the openshift-marketplace as follows:

### 1. Create the namespace
```console
$ cat <<EOF | oc apply -f -
apiVersion: v1
kind: Namespace
metadata:
  name: openshift-logging
---
EOF
```
### 2. Create the operator group
```console
$ cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: cluster-logging
  namespace: openshift-logging 
spec:
  targetNamespaces:
  - openshift-logging
---
EOF
```
### 3. Create the subscription
```console
$ cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: cluster-logging
  namespace: openshift-logging 
spec:
  channel: "5.0" 
  name: cluster-logging
  source: redhat-operators 
  sourceNamespace: openshift-marketplace
---
EOF
```
## Get Kafka broker details
To ship logs to Kafka you need the following details for configuration of your `outputs`:
- Broker IP and port
- Topic(s) name(s)
- Security settings

For the demonstration purposes you can deploy your own Kafka instance on a VM as described in [Appendix: Kafka test broker deployment](#kafka_deploy)

## Configure ClusterLogging resource
```console
$ cat <<EOF | oc apply -f -
apiVersion: logging.openshift.io/v1
kind: ClusterLogging
metadata:
  name: instance 
  namespace: openshift-logging
spec:
  managementState: "Managed"
  collection:
    logs:
      type: "fluentd"  
      fluentd: {}
---
EOF
```
This will deploy fluentd collector on the node (in multi-node clusters - on every node)

## Configure ClusterLog Forwarder resource
```console
$ cat <<EOF | oc apply -f -
apiVersion: "logging.openshift.io/v1"
kind: ClusterLogForwarder
metadata:
  name: instance 
  namespace: openshift-logging 
spec:
  outputs:
  - type: "kafka"
    name: kafka-open
    url: tcp://10.46.55.190:9092/test
  inputs: 
  - name: infra-logs
    infrastructure:
      namespaces:
      - openshift-apiserver
      - openshift-cluster-version
      - openshift-etcd
      - openshift-kube-scheduler
      - openshift-monitoring
  pipelines:
  - name: audit-logs 
    inputRefs:
    - audit
    outputRefs:
    - kafka-open
  - name: infrastructure-logs 
    inputRefs:
    - infrastructure
    outputRefs:
    - kafka-open
---
EOF
```
This will configure your fluentd to collect the logs from the specified namespaces and ship them to the Kafka bus defined in the `outputs`: tcp://10.46.55.190:9092, topic `test`.

## Appendix: Kafka test broker deployment<a name="kafka_deploy"></a>
### 1. Get a VM
This can be done with kcli:
```console
$ kcli create vm -i centos8 -P memory=4096 -P disks=[200] -P nets=[baremetal] -P cmds=["yum -y install java wget"] kafka
```
### 2. Download and extract Kafka
Go to this link to find the best mirror:
https://www.apache.org/dyn/closer.cgi?path=/kafka/2.7.0/kafka_2.12-2.7.0.tgz
and then download kafka_2.12-2.7.0.tgz, for example:
```console
$ wget https://apache.mivzakim.net/kafka/2.7.0/kafka_2.12-2.7.0.tgz
$ tar -xf kafka_2.12-2.7.0.tgz
```
### 3. Configure the message size
Default Kafka message size is smaller than the fluentd message size. Change it as follows:
```console
$ cd kafka_2.12-2.7.0
$ echo "message.max.bytes=10485760" >> config/server.properties
```
### 4. Start zookeeper
```console
$ bin/zookeeper-server-start.sh config/zookeeper.properties &
```
Wait for Zookeeper to start

### 5. Start Kafka
```console
$ bin/kafka-server-start.sh config/server.properties &
```
### 6. Create the `test` topic
```console
$ bin/kafka-topics.sh --create --bootstrap-server 127.0.0.1:9092 --replication-factor 1 --partitions 1 --topic test
```

### 7. Start the console consumer
```console
$ bin/kafka-console-consumer.sh --bootstrap-server 10.46.55.190:9092 --topic test
```
Your logs should be visible in the console after you configure the ClusterLogForwarder resource above
For troubleshooting it helps to start a console producer in a separate window (or on another machine):

```console
$ bin/kafka-console-producer.sh --bootstrap-server 10.46.55.190:9092 --topic test
```

Now you can send messages to your console consumer to verify that the broker works


