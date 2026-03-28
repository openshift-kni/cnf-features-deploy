# Getting started

Thanks for asking on how to contribute! Let's get you started!

## GitHub workflow

To check out code to work on, please refer to the [Kubernetes GitHub Workflow
Guide](https://www.kubernetes.dev/docs/guide/github-workflow/)

Besides of what's written in the guide, we encourage you to:

* Keep atomic PRs when feasible
* Squash/Rebase on the go, it will make reviewers' life easier!
* Split big PRs into small ones

Before hitting the `submit` button, please also ensure that all commits contain
a well written description including a title and a description.

### Learning Git

You can use [Git Immersion](https://gitimmersion.com/lab_02.html) in case you
need some tutorial to gain more knowledge regarding git.

For reference, use the [Git Reference and Cheat Sheet](https://git-scm.com/doc)

#### Commit Messages

Commit messsages are the first things a reviewer sees and are used as
descriptions in the git log. They provide a description of the history of
changes in a repository. Commit messages cannot be modified once the patch is
merged.

Format:

* Summary Line
* Empty line
* Body

##### Summary Line

The summary line briefly describes the patch content. The character limit is 50
characters. The summary line should not end with a period. If the change is not
finished at the time of the commit, start the commit message with WIP.

Specifically for openshift-kni/cnf-features-deploy and ztp, the summary line
must start with `ztp:` or your changes will be refused by CI.

##### Body

The body contains the explanation of the issue being solved and why it should
be fixed, the description of the solution, and additional optional information
on how it improves the code structure, or references to other relevant patches,
for example. The lines are limited to 72 characters. The body should contain
all the important information related to the problem, without assuming that the
reader understands the source of the problem or has access to external sites.

## Coding Conventions

You can follow the Code Conventions from Kubernetes [here](https://www.kubernetes.dev/docs/guide/coding-convention/)

## Kubernetes/OpenShift CI

You can learn on how to interact with Prow [here](https://prow.k8s.io/command-help)

## Kubernetes Contributor Playground

If you are looking for a safe place, where you can familiarize yourself with
the pull request and issue review process in Kubernetes, then the [Kubernetes
Contributor Playground](https://github.com/kubernetes-sigs/contributor-playground/blob/master/README.md)
is the right place for you.
