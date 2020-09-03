#!/bin/bash
set -e

. $(dirname "$0")/common.sh

export SUITES_PATH=cnf-tests/bin

# generate the junit files
hack/build-test-bin.sh
cnf-tests/test-run.sh -junit _cache/junit -ginkgo.dryRun -ginkgo.v

go build -o _cache/docgen cnf-tests/docgen/main.go 
# use the junit files to fill the descriptions
_cache/docgen fill --junit _cache/junit/cnftests-junit.xml --description cnf-tests/docgen/e2e.json
_cache/docgen fill --junit _cache/junit/validation_junit.xml --description cnf-tests/docgen/validation.json
