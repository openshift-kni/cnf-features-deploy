#!/bin/bash
set -e

. $(dirname "$0")/common.sh

export SUITES_PATH=bin

# generate the junit files
hack/build-test-bin.sh
entrypoint/test-run.sh -junit _cache/junit --ginkgo.dry-run -ginkgo.v

go build -o _cache/docgen docgen/main.go 
# use the junit files to fill the descriptions
_cache/docgen fill --junit _cache/junit/junit_cnftests.xml --description docgen/e2e.json
_cache/docgen fill --junit _cache/junit/junit_validation.xml --description docgen/validation.json
