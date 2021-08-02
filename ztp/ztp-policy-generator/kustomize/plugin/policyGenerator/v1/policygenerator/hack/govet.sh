#!/bin/bash

. $(dirname $0)/common.sh
echo "INFO: Running go vet on ztp policy-generator"
go vet ./...
