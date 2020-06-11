#!/usr/bin/env bash
set -e

make -e -C test-pmd

cp test-pmd/testpmd ./customtestpmd

echo "build done"
