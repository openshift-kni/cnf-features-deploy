# Gitops subscription for ArgoCD and ACM 

This directory contain the subscription CRs for ArgoCD and ACM.

- Argocd

  We recommend using the [Red Hat OpenShift GitOps operator](https://catalog.redhat.com/software/operators/detail/5fb288c70a12d20cbecc6056) to deploy ArgoCD operator. Use argocd/deployment/kustomization.yaml in order to apply the Argocd CRs in the right order.

- ACM

  The ACM subscription CRs required the ACM operator installed in the hub cluster.
