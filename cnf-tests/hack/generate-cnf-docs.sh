#!/bin/bash

. $(dirname "$0")/common.sh

go build  -o _cache/docgen docgen/main.go
_cache/docgen generate --target TESTLIST.md --testsjson docgen/e2e.json --validationjson docgen/validation.json
