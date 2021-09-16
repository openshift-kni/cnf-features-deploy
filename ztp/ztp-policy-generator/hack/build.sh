#!/bin/bash

. $(dirname $0)/common.sh

cd "${PLUGINPATH}"
go build -o $PLUGINBIN
