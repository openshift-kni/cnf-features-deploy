#!/usr/bin/env bash
set -e

make -C test-pmd

cp test-pmd/testpmd ./customtestpmd

echo "build done"
