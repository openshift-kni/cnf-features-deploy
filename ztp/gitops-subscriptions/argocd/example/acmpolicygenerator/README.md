## ACM PolicyGenerator
The [ACM PolicyGenerator](https://github.com/stolostron/policy-generator-plugin) examples under [here](https://github.com/openshift-kni/cnf-features-deploy/tree/master/ztp/gitops-subscriptions/argocd/example/acmpolicygenerator) are defining the DU profile policies using [ACM PolicyGenerator reference API](https://github.com/stolostron/policy-generator-plugin/blob/main/docs/policygenerator-reference.yaml). These examples will generate ACM policies same as the DU profile policies generated from [policygentemplates example](https://github.com/openshift-kni/cnf-features-deploy/tree/master/ztp/gitops-subscriptions/argocd/example/policygentemplates) AND specifically the policy content object-definition (source-crs) are identical.

##### Setup ArgoCD Application

The DU profile ACM PolicyGenerator examples can be used with the same [policies-app](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/gitops-subscriptions/argocd/deployment/policies-app.yaml) that is used to deploy the DU profile policygentemplates examples. For more info how to setup the ArgoCD policies application follow the [ReadMe](https://github.com/openshift-kni/cnf-features-deploy/blob/master/ztp/gitops-subscriptions/argocd/README.md) section "Preparation of Hub cluster for ZTP". The Git repo that will be used with the ArgoCD policies application should contain the source-crs directory and must co-exist with the DU profile ACM PolicyGenerator as shown below as example

```
├── acmpolicygenerator
│   ├── source-crs/
│   │    ├── AcceleratorsNS.yaml
│   │    ├── ...
├── acm-common-ranGen.yaml
├── acm-example-multinode-site.yaml
├── acm-example-sno-site.yaml
├── acm-group-du-3node-ranGen.yaml
├── acm-group-du-3node-validator-ranGen.yaml
├── acm-group-du-sno-ranGen.yaml
├── acm-group-du-sno-validator-ranGen.yaml
├── acm-group-du-standard-ranGen.yaml
├── acm-group-du-standard-validator-ranGen.yaml
├── kustomization.yaml
├── ns.yaml

```

For more info using PolicyGenerator follow the [ACM PolicyGenerator examples](https://github.com/stolostron/policy-generator-plugin/tree/main/examples).
