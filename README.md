# Overview

This repo contains example kustomize configs used to installed openshift features required for CNF workloads and a e2e functional test suite used to verify cnf related features.

# Contributing kustomize configs

All kustomize configs should be entirely declarative in nature. This means no bash plugin modules performing imparative tasks. Features should be installed simply by posting manifests to the cluster. After posting manifests, determining when the cluster has converged on those manifests successully should be observable. 
