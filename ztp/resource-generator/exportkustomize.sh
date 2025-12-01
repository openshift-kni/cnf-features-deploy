#!/usr/bin/env bash

# This shell script is an entrypoint in the container used to patch ArgoCD kustomize with the PolicyGenTemplate plugins.

destPath=$1

cp -r /kustomize $destPath
