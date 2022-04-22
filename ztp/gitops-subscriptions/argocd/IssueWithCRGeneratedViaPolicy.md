## Duplicated list entries when modifying Policy

If you're using the default configurations and update a CR's list via PolicyGenTemplate,  you may notice that the "final" CR generated doesn't match the Policy.

e.g After updating `TunedPerformancePatch.yaml`'s `data` list, the final `tuned` CR maybe look like the one below.

```yaml
[core@cnfdf02 ~]$ oc get tuned -n openshift-cluster-node-tuning-operator performance-patch -o yaml
apiVersion: tuned.openshift.io/v1
kind: Tuned
metadata:
  name: performance-patch
  namespace: openshift-cluster-node-tuning-operator
spec:
  profile:
    - data: |
        ... # rest of data

        [sysctl]
        kernel.timer_migration=1
      name: performance-patch
    - data: |
        ...  # rest of data

        [sysctl]
        kernel.timer_migration=0 # in PGT this is the value that was updated. But now there's a new list entry.
      name: performance-patch
```

But `Policy` is reporting is what is expected, i.e list of `data` was replaced instead of appended. 

```yaml
[core@cnfdf02 ~]$ oc get policy group-cnfdf02.group-cnfdf02-tuned-policy -n cnfdf02 -o yaml
  ...
        object-templates:
        - complianceType: musthave
          objectDefinition:
            apiVersion: tuned.openshift.io/v1
            kind: Tuned
            metadata:
              name: performance-patch
              namespace: openshift-cluster-node-tuning-operator
            spec:
              profile:
              - data: |
                  [main]
                  summary=Configuration changes profile inherited from performance created tuned
                  include=openshift-node-performance-openshift-node-performance-profile

                  [sysctl]
                  kernel.timer_migration=0
                name: performance-patch
  ...
```

## How to fix

The behaviour is actually coming from ACM's `configuration-policy-controller` and is expected. 
To work-around it just at `mustonlyhave` to the file under your PGT. E.g 

```yaml
- fileName: TunedPerformancePatch.yaml
  complianceType: mustonlyhave
```

## Impact on CPU and system

Adding `mustonlyhave` didn't have any impact on CPU consumption in this case, but it's recommended to use `mustonlyhave` on an as-needed basis. If added keep an eye on the CPU consumption. A useful PromQL query for this below:
```shell
avg_over_time(pod:container_cpu_usage:sum{namespace=~"openshift-logging|openshift-authentication-operator|openshift-kube-controller-manager|openshift-kube-apiserver|openshift-apiserver|open-cluster-management.*"}[30m:30s])
```
Higher consumption may also be an indicator of a "race condition" where two operators (policy-controller from ACM vs another-operator) are both constantly trying to update the same CR. This may also have unintended consequences such as deployment failure.