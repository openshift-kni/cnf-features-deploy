#!/bin/bash
set -e

. $(dirname "$0")/common.sh

go build -o _cache/docgen docgen/main.go

# generate the junit files
TESTS_REPORTS_PATH=cnf-tests/_cache/junit GINKGO_PARAMS=--dry-run ../hack/run-functests.sh

# use the junit files to fill the descriptions
_cache/docgen fill --junit _cache/junit/junit_cnftests.xml --description docgen/e2e.json
_cache/docgen fill --junit _cache/junit/junit_validation.xml --description docgen/validation.json
