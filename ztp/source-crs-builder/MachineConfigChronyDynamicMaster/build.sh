#!/bin/bash

GOPATH=${GOPATH:-${HOME}/go}
GOBIN=${GOBIN:-${GOPATH}/bin}
MCMAKER=${MCMAKER:-${GOBIN}/mcmaker}

${MCMAKER} -stdout -name 05-chronyd-dynamic -mcp master \
	file -source ptp-sync-check -path /usr/local/bin/ptp-sync-check -mode 0755 \
	file -source restart-chronyd -path /usr/local/bin/restart-chronyd -mode 0755 \
	unit -source chronyd-restart.service -enable=false \
    unit -source chronyd-restart.timer \
	dropin -source 20-conditional-start.conf -for chronyd.service	

