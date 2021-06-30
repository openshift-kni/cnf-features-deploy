#!/usr/bin/env bash
set -e

make static -e -C test-pmd

cp test-pmd/build/testpmd-static ./customtestpmd

echo "build done"
