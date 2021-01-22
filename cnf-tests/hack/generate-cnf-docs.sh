#!/bin/bash

. $(dirname "$0")/common.sh

go build  -o _cache/docgen cnf-tests/docgen/main.go
_cache/docgen generate --target cnf-tests/TESTLIST.md --testsjson cnf-tests/docgen/e2e.json --validationjson cnf-tests/docgen/validation.json
