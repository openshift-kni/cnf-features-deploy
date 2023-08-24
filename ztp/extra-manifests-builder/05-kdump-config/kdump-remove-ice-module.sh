#!/usr/bin/env bash

# This script removes the ice module from kdump to prevent kdump failures on certain servers.
# This is a temporary workaround for RHELPLAN-138236 and can be removed when that issue is
# fixed.

set -x

SED="/usr/bin/sed"
GREP="/usr/bin/grep"

# override for testing purposes
KDUMP_CONF="${1:-/etc/sysconfig/kdump}"
REMOVE_ICE_STR="module_blacklist=ice"

# exit if file doesn't exist
[ ! -f ${KDUMP_CONF} ] && exit 0

# exit if file already updated
${GREP} -Fq ${REMOVE_ICE_STR} ${KDUMP_CONF} && exit 0

# Target line looks something like this:
# KDUMP_COMMANDLINE_APPEND="irqpoll nr_cpus=1 ... hest_disable"
# Use sed to match everything between the quotes and append the REMOVE_ICE_STR to it
${SED} -i 's/^KDUMP_COMMANDLINE_APPEND="[^"]*/& '${REMOVE_ICE_STR}'/' ${KDUMP_CONF} || exit 0
