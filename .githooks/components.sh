#!/bin/bash

declare -A components
components=( \
	["tools"]="tools" \
	["s2i"]="tools/s2i-dpdk" \
	["features"]="feature-configs" \
	["ztp"]="ztp","go.mod","go.sum" \
	["infra"]="Makefile",".githooks/","hack/" \
	["vendor"]="vendor","cnf-tests","ztp" \
	["cnf-tests"]="cnf-tests","go.mod","go.sum"
)

