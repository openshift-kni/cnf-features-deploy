#!/bin/bash

. $(dirname "$0")/common.sh

gofmt_command="gofmt -s -l `find . -type d \( -path ./vendor -o -path ./cnf-tests/submodules \) -prune -o -type f -name '*.go' -print`"
eval $gofmt_command
if [[ -z $(eval ${gofmt_command}) ]]; then
	echo "INFO: gofmt success"
	exit 0
else
	echo "ERROR: gofmt reported formatting issues"
	exit 1
fi
