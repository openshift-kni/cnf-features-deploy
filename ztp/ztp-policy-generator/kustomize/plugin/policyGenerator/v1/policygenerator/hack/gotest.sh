#!/bin/bash

. $(dirname $0)/common.sh
echo "INFO: Running go test on ztp policy-generator"
go test ./...
