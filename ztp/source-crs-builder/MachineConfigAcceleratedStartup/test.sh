#!/bin/bash

fatal() {
  echo "FATAL: $@"
  exit 1
}

echo "Testing import..."
. ./accelerated-container-startup.sh
rc=$?
[[ $rc -eq 0 ]] || fatal "Could not import"
echo Ok

test_within() {
  local expected=$1; shift
  within "$@"
  rc=$?
  [[ $rc -eq $expected ]] || fatal "within failed: Expected rc $rc != $expected"
}

test_steadystate() {
  local expected=$1
  STEADY_STATE_THRESHOLD=$4
  steadystate $2 $3
  rc=$?
  [[ $rc -eq $expected ]] || fatal "steadystate failed: Expected rc $rc != $expected"
}

# Trivial case: 0->0 = 0% change
test_within 1 0 0 -1
test_within 0 0 0 0
test_within 0 0 0 0%
test_within 0 0 0 1
test_within 0 0 0 1%
test_within 0 0 0 2
test_within 0 0 0 2%

# Edge case: 0->1 = undefined change (treated as very big change)
test_within 1 0 1 -1
test_within 1 0 1 0
test_within 1 0 1 0%
test_within 0 0 1 1
test_within 1 0 1 1%
test_within 0 0 1 2
test_within 1 0 1 2%

# Edge case: 1->0 = 100% change but a small delta
test_within 1 1 0 -1
test_within 1 1 0 0
test_within 1 1 0 0%
test_within 0 1 0 1
test_within 1 1 0 1%
test_within 0 1 0 2
test_within 1 1 0 2%

# Steady-state: 1->1 = 0% change
test_within 1 1 1 -1
test_within 0 1 1 0
test_within 0 1 1 0%
test_within 0 1 1 1
test_within 0 1 1 1%
test_within 0 1 1 2
test_within 0 1 1 2%

# Mixed case: 1->2 = 50% change, but a small delta
test_within 1 1 2 -1
test_within 1 1 2 0
test_within 1 1 2 0%
test_within 0 1 2 1
test_within 1 1 2 1%
test_within 0 1 2 2
test_within 1 1 2 2%

# Steady-state: 100->100 = 0% change
test_within 1 100 100 -1
test_within 0 100 100 0
test_within 0 100 100 0%
test_within 0 100 100 1
test_within 0 100 100 1%
test_within 0 100 100 2
test_within 0 100 100 2%

# Negative small change: 101->100 = 0% change
test_within 1 101 100 -1
test_within 1 101 100 0
test_within 0 101 100 0%
test_within 0 101 100 1
test_within 0 101 100 1%
test_within 0 101 100 2
test_within 0 101 100 2%

# Negative small change: 102->100 = 1% change
test_within 1 102 100 -1
test_within 1 102 100 0
test_within 1 102 100 0%
test_within 1 102 100 1
test_within 0 102 100 1%
test_within 0 102 100 2
test_within 0 102 100 2%

# Test with/without allowed-zero
echo "Allowing last >= 0"
STEADY_STATE_MINIMUM=0
test_steadystate 0 0 0 0
test_steadystate 0 1 0 1
test_steadystate 0 2 3 2
echo "Only allowing last >= 2"
STEADY_STATE_MINIMUM=2
test_steadystate 1 0 0 0
test_steadystate 1 1 0 1
test_steadystate 0 2 3 2

echo "All tests completed successfully!"
