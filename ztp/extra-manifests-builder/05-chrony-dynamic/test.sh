#!/bin/bash

fatal() {
  echo "FATAL: $@"
  exit 1
}

echo "Testing import..."
. ./ptp-sync-check
rc=$?
[[ $rc -eq 0 ]] || fatal "Could not import"
echo Ok

# Redefine functions that use system facilities
get_last_ptp_status(){
  echo $MOCK
  return 0
}

log_debug() {
  :
}

mock(){
  mock_const_part="stdout F ptp4l[8180.150]: [ens4f0] master offset"
  MOCK="$1 $mock_const_part $2"
}

test_get_ptp_offset() {
  local expected_rc=$1; 
  local expected_offset=$2
  offset=$(get_ptp_offset)
  rc=$?
  [[ $rc -eq $expected_rc ]] || fatal "test_get_ptp_offset failed: Expected rc $expected_rc != $rc"
  [[ $offset -eq $expected_offset ]] || fatal "test_get_ptp_offset failed: Expected offset $expected_offset != $offset"
}

current_date=$(date -Ins)
current_date=${current_date//,/.}
expired_date=$(date -d "-15 minutes" -Ins)
expired_date=${expired_date//,/.}

echo "Testing get_ptp_offset..."

mock $current_date "-8"
test_get_ptp_offset 0 8
mock $expired_date "5"
test_get_ptp_offset 1 5
