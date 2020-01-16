#!/bin/bash

which go
if [ $? -ne 0 ]; then
  echo "No go command available"
  exit 1
fi

GOPATH="${GOPATH:-~/go}"
export PATH=$PATH:$GOPATH/bin

export failed=false
export failures=()

if [ "$FEATURES" == "" ]; then
	echo "[ERROR]: No FEATURES provided"
	exit 1
fi

which ginkgo
if [ $? -ne 0 ]; then
	echo "Downloading ginkgo tool"
	go install github.com/onsi/ginkgo/ginkgo
fi


FOCUS=$(echo "$FEATURES" | tr ' ' '|') 
echo "Focusing on $FOCUS"
if ! GOFLAGS=-mod=vendor ginkgo --focus=$FOCUS functests -- -junit /tmp/artifacts/unit_report.xml; then
  failed=true
  failures+=( "Tier 2 tests for $FEATURES" )
fi

for feature in $FEATURES; do
  test_entry_point=external-tests/${feature}/test.sh
  if [[ ! -f $test_entry_point ]]; then
    echo "[INFO] Feature '$feature' does not have external tests entry point"
    continue
  fi
  echo "[INFO] Running external tests for $feature"
  set +e
  if ! $test_entry_point; then
    failures+=( "Tier 1 tests for $feature" )
    failed=true
  fi
  set -e
done

if $failed; then
  echo "[WARN] Tests failed:"
  for failure in "${failures[@]}"; do
    echo "$failure"
  done;
  exit 1
fi
