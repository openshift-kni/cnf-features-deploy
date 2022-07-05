#!/usr/bin/env bash
set -e

cd l2fwd

make

cp build/l2fwd-shared ../dpdk-l2fwd

echo "build done"
