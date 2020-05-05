#!/bin/sh

oc -n dpdk get build -l app=s2i-dpdk

oc -n dpdk wait build -l app=s2i-dpdk --for condition=Complete --timeout 1s

