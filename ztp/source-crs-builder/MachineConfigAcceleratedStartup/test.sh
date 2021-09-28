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
  [[ $rc -eq $expected ]] || fatal "within failed: Expected rc $expected != $rc"
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

test_steadystate() {
  local expected=$1
  STEADY_STATE_THRESHOLD=$4
  steadystate $2 $3
  rc=$?
  [[ $rc -eq $expected ]] || fatal "steadystate failed: Expected rc $expected != $rc"
}

# Test with minimum boundary
echo "Allowing minimum >= 0"
STEADY_STATE_MINIMUM=0
test_steadystate 0 0 0 0
test_steadystate 0 1 0 1
test_steadystate 0 2 3 2
echo "Only allowing minimum >= 2"
STEADY_STATE_MINIMUM=2
test_steadystate 1 0 0 0
test_steadystate 1 1 0 1
test_steadystate 0 2 3 2

test_unrestrictedCpuset() {
    local expectedRc="$1" expectedValue="$2" kubeletContent="$3" defaultState="$4"
    echo "Testing unrestrictedCpuset: $5"
    KUBELET_CPU_STATE=/tmp/$$-kubelet-state
    FULL_CPU_STATE=/tmp/$$-default-state
    if [[ -n $kubeletContent ]]; then
        touch $KUBELET_CPU_STATE
        if [[ $kubeletContent != ' ' ]]; then
            echo "$kubeletContent" > $KUBELET_CPU_STATE
        fi
    fi
    if [[ -n $defaultState ]]; then
        echo "$defaultState" > $FULL_CPU_STATE
    fi
    local result
    result=$(unrestrictedCpuset)
    local rc=$?
    rm -f $KUBELET_CPU_STATE $FULL_CPU_STATE
    [[ $rc -eq $expectedRc ]] || fatal "  unrestrictedCpuset failed: Expected rc $expectedRc != $rc"
    [[ $result == $expectedValue ]] || fatal "  unrestrictedCpuset failed: Expected return value '$expectedValue' != '$result'"
}

test_unrestrictedCpuset 0 "4-5" '{ "defaultCpuSet": "4-5" }' "0-9" "Valid parse results"
test_unrestrictedCpuset 0 "0-9" '{ INVALID JSON t": "4-5" }' "0-9" "Invalid json fallback"
test_unrestrictedCpuset 0 "0-9" ' '                          "0-9" "Empty file fallback"
test_unrestrictedCpuset 0 "0-9" ''                           "0-9" "Missing file falback"
test_unrestrictedCpuset 1 ''    ''                           ''    "Fallback file missing"

echo "All tests completed successfully!"
