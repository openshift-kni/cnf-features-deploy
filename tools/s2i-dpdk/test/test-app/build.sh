#!/usr/bin/env bash

make -C test-pmd

cp test-pmd/testpmd ./customtestpmd

echo "build done"
