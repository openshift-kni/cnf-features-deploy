#!/bin/bash

. $(dirname "$0")/common.sh

MDFILE="TESTLIST.md"

mv cnf-tests/${MDFILE} _cache/${MDFILE}.old

. $(dirname "$0")/generate-cnf-docs.sh

diff cnf-tests/${MDFILE} _cache/${MDFILE}.old -q || { echo "Docs should be regenerated and updated upstream. You can use hack/fill-empty-docs.sh and hack/generate-cnf-docs.sh"; exit 1; }
