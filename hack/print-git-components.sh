#!/bin/bash
. $(dirname "$0")/common.sh
set +e

. $(dirname "$0")/../.githooks/components.sh

echo components for git commits are:
for component in "${!components[@]}"; do
	printf "  %-10s -> %-20s\n" $component ${components[$component]}
done
