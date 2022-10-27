# K8s Daemonset in privilege mode

## Functions

- Creation of the daemonset in priviledge mode
- Deletion of a daemonset in a specified namespace
- Check if a daemonset is ready within a specified time


## Usage

1. Import the library

```
import k8sPriviledgedDs "github.com/test-network-function/privileged-daemonset"
```

2. Set the K8s client to act on `Daemonset` object

``` 
k8sPriviledgedDs.SetDaemonSetClient(myK8sInterface) // myK8sInterface is of type kubernetes.Interface
```

3. Invoke the exported functions in a specified namespace with a specified imagename

**To create**

``` 
daemonSetRunningPods, err := k8sPriviledgedDs.CreateDaemonSet(myDaemonSetName, myNameSpace, daemonSetContainerName, imageWithVersion, timeOut)
```

**To delete**

``` 
err := k8sPriviledgedDs.DeleteDaemonSet(myDaemonSetName, myNameSpace)
```

**To check if the daemonset is ready**

``` 
err := k8sPriviledgedDs.WaitDaemonsetReady(myDaemonSetName, myNameSpace)
```
