#!/bin/bash

which go
if [ $? -ne 0 ]; then
  echo "No go command available"
  exit 1
fi

GOPATH="${GOPATH:-~/go}"
export PATH=$PATH:$GOPATH/bin

which ginkgo
if [ $? -ne 0 ]; then
	echo "Downloading ginkgo tool"
	go install github.com/onsi/ginkgo/ginkgo
fi


FOCUS=$(echo "$FEATURES" | tr ' ' '|') 
echo "Focusing on $FOCUS"
GOFLAGS=-mod=vendor ginkgo --focus=$FOCUS functests -- -junit /tmp/artifacts/unit_report.xml
