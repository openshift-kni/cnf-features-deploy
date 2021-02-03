#!/bin/bash

. $(dirname "$0")/common.sh

MDFILE="TESTLIST.md"

mv ${MDFILE} _cache/${MDFILE}.old

hack/generate-cnf-docs.sh

diff ${MDFILE} _cache/${MDFILE}.old -q || { echo "Docs should be regenerated and updated upstream. You can use cnf-tests/hack/fill-empty-docs.sh and cnf-tests/hack/generate-cnf-docs.sh"; exit 1; }
