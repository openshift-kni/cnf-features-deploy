#!/bin/bash

GOPATH=${GOPATH:-${HOME}/go}
GOBIN=${GOBIN:-${GOPATH}/bin}
MCMAKER=${MCMAKER:-${GOBIN}/mcmaker}
MCPROLE=${MCPROLE:-master}

${MCMAKER} -stdout -name 08-set-rcu-normal -mcp ${MCPROLE} \
	file -source set-rcu-normal.sh -path /usr/local/bin/set-rcu-normal.sh -mode 0755 \
	unit -source set-rcu-normal.service
