#!/bin/bash

declare -A components
components=( \
	["tools"]="tools" \
	["s2i"]="tools/s2i-dpdk" \
	["features"]="feature-configs" \
	["ztp"]="ztp" \
	["infra"]="Makefile",".githooks/","hack/","openshift-ci/" \
	["vendor"]="vendor","cnf-tests","ztp","go.mod","go.sum" \
	["cnf-tests"]="cnf-tests" \
	["oot-driver"]="tools/oot-driver" \
	["owners"]="OWNERS","OWNERS_ALIASES" \
	["docs"]="README.md"
)

