#!/bin/bash

. $(dirname "$0")/common.sh
echo "INFO: Running govet on cnf-tests/testsuites/..."
go vet ./testsuites/...
