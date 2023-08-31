#!/bin/bash
#
# Disable rcu_expedited after node has finished booting
#
# The defaults below can be overridden via environment variables
#

# Default wait time is 600s = 10m:
MAXIMUM_WAIT_TIME=${MAXIMUM_WAIT_TIME:-600}

# Default steady-state threshold = 2%
# Allowed values:
#  4  - absolute pod count (+/-)
#  4% - percent change (+/-)
#  -1 - disable the steady-state check
STEADY_STATE_THRESHOLD=${STEADY_STATE_THRESHOLD:-2%}

# Default steady-state window = 60s
# If the running pod count stays within the given threshold for this time
# period, return CPU utilization to normal before the maximum wait time has
# expires
STEADY_STATE_WINDOW=${STEADY_STATE_WINDOW:-60}

# Default steady-state allows any pod count to be "steady state"
# Increasing this will skip any steady-state checks until the count rises above
# this number to avoid false positives if there are some periods where the
# count doesn't increase but we know we can't be at steady-state yet.
STEADY_STATE_MINIMUM=${STEADY_STATE_MINIMUM:-0}

#######################################################

within() {
  local last=$1 current=$2 threshold=$3
  local delta=0 pchange
  delta=$(( current - last ))
  if [[ $current -eq $last ]]; then
    pchange=0
  elif [[ $last -eq 0 ]]; then
    pchange=1000000
  else
    pchange=$(( ( "$delta" * 100) / last ))
  fi
  echo -n "last:$last current:$current delta:$delta pchange:${pchange}%: "
  local absolute limit
  case $threshold in
    *%)
      absolute=${pchange##-} # absolute value
      limit=${threshold%%%}
      ;;
    *)
      absolute=${delta##-} # absolute value
      limit=$threshold
      ;;
  esac
  if [[ $absolute -le $limit ]]; then
    echo "within (+/-)$threshold"
    return 0
  else
    echo "outside (+/-)$threshold"
    return 1
  fi
}

steadystate() {
  local last=$1 current=$2
  if [[ $last -lt $STEADY_STATE_MINIMUM ]]; then
    echo "last:$last current:$current Waiting to reach $STEADY_STATE_MINIMUM before checking for steady-state"
    return 1
  fi
  within "$last" "$current" "$STEADY_STATE_THRESHOLD"
}

waitForReady() {
  logger "Recovery: Waiting ${MAXIMUM_WAIT_TIME}s for the initialization to complete"
  local t=0 s=10
  local lastCcount=0 ccount=0 steadyStateTime=0
  while [[ $t -lt $MAXIMUM_WAIT_TIME ]]; do
    sleep $s
    ((t += s))
    # Detect steady-state pod count
    ccount=$(crictl ps 2>/dev/null | wc -l)
    if [[ $ccount -gt 0 ]] && steadystate "$lastCcount" "$ccount"; then
      ((steadyStateTime += s))
      echo "Steady-state for ${steadyStateTime}s/${STEADY_STATE_WINDOW}s"
      if [[ $steadyStateTime -ge $STEADY_STATE_WINDOW ]]; then
        logger "Recovery: Steady-state (+/- $STEADY_STATE_THRESHOLD) for ${STEADY_STATE_WINDOW}s: Done"
        return 0
      fi
    else
      if [[ $steadyStateTime -gt 0 ]]; then
        echo "Resetting steady-state timer"
        steadyStateTime=0
      fi
    fi
    lastCcount=$ccount
  done
  logger "Recovery: Recovery Complete Timeout"
}

setRcuNormal() {
  echo "Setting rcu_normal to 1"
  echo 1 > /sys/kernel/rcu_normal
}

main() {
  waitForReady
  echo "Waiting for steady state took: $(awk '{print int($1/3600)"h", int(($1%3600)/60)"m", int($1%60)"s"}' /proc/uptime)"
  setRcuNormal
}

if [[ "${BASH_SOURCE[0]}" = "${0}" ]]; then
  main "${@}"
  exit $?
fi
