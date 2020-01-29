#!/bin/sh

oc -n dpdk-testing get build -l app=s2i-dpdk

oc -n dpdk-testing wait build -l app=s2i-dpdk --for condition=Complete --timeout 1s

oc get dc s2i-dpdk-app

oc -n dpdk-testing wait dc -l app=s2i-dpdk-app --for condition=Available --timeout 1s
