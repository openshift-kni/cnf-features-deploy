#!/bin/bash
#
# Temporarily reset the core system processes's CPU affinity to be unrestricted to accelerate startup and shutdown
#
# The defaults below can be overridden via environment variables
#

# The default set of critical processes whose affinity should be temporarily unbound:
CRITICAL_PROCESSES=${CRITICAL_PROCESSES:-"systemd ovs crio kubelet NetworkManager conmon dbus"}

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

unrestrictedCpuset() {
  if [[ ! -e /var/lib/kubelet/cpu_manager_state ]]; then
    # use all the cpus if kubelet is not configured yet
    cat /sys/fs/cgroup/cpuset/cpuset.cpus
  else
    jq -r '.defaultCpuSet' </var/lib/kubelet/cpu_manager_state
  fi
}

restrictedCpuset() {
  for arg in $(</proc/cmdline); do
    if [[ $arg =~ ^systemd.cpu_affinity= ]]; then
      echo ${arg#*=}
      return 0
    fi
  done
  return 1
}

resetAffinity() {
  local cpuset="$1"
  local failcount=0
  local successcount=0
  logger "Recovery: Setting CPU affinity for critical processes \"$CRITICAL_PROCESSES\" to $cpuset"
  for proc in $CRITICAL_PROCESSES; do
    local pids="$(pgrep $proc)"
    for pid in $pids; do
      local tasksetOutput
      tasksetOutput="$(taskset -apc "$cpuset" $pid 2>&1)"
      if [[ $? -ne 0 ]]; then
        echo "ERROR: $tasksetOutput"
        ((failcount++))
      else
        ((successcount++))
      fi
    done
  done
  logger "Recovery: Re-affined $successcount pids successfully"
  if [[ $failcount -gt 0 ]]; then
    logger "Recovery: Failed to re-affine $failcount processes"
    return 1
  fi
}

setUnrestricted() {
  logger "Recovery: Setting critical system processes to have unrestricted CPU access"
  resetAffinity "$(unrestrictedCpuset)"
}

setRestricted() {
  logger "Recovery: Resetting critical system processes back to normally restricted access"
  resetAffinity "$(restrictedCpuset)"
}

currentAffinity() {
  local pid="$1"
  taskset -pc $pid | awk -F': ' '{print $2}'
}

within() {
  local last=$1 current=$2 threshold=$3
  local delta=0 pchange
  delta=$(( current - last ))
  if [[ $current -eq $last ]]; then
    pchange=0
  elif [[ $last -eq 0 ]]; then
    pchange=1000000
  else
    pchange=$(( ( $delta * 100) / last ))
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
  within $last $current $STEADY_STATE_THRESHOLD
}

waitForReady() {
  logger "Recovery: Waiting ${MAXIMUM_WAIT_TIME}s for the initialization to complete"
  local lastSystemdCpuset="$(currentAffinity 1)"
  local lastDesiredCpuset="$(unrestrictedCpuset)"
  local t=0 s=10
  local lastCcount=0 ccount=0 steadyStateTime=0
  while [[ $t -lt $MAXIMUM_WAIT_TIME ]]; do
    sleep $s
    ((t += s))
    # Re-check the current affinity of systemd, in case some other process has changed it
    local systemdCpuset="$(currentAffinity 1)"
    # Re-check the unrestricted Cpuset, as the allowed set of unreserved cores may change as pods are assigned to cores
    local desiredCpuset="$(unrestrictedCpuset)"
    if [[ $systemdCpuset != $lastSystemdCpuset || $lastDesiredCpuset != $desiredCpuset ]]; then
      resetAffinity "$desiredCpuset"
      lastSystemdCpuset="$(currentAffinity 1)"
      lastDesiredCpuset="$desiredCpuset"
    fi

    # Detect steady-state pod count
    ccount=$(crictl ps | wc -l)
    if steadystate $lastCcount $ccount; then
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

main() {
  if ! unrestrictedCpuset >&/dev/null; then
    logger "Recovery: No unrestricted Cpuset could be detected"
    return 1
  fi

  if ! restrictedCpuset >&/dev/null; then
    logger "Recovery: No restricted Cpuset has been configured.  We are already running unrestricted."
    return 0
  fi

  # Ensure we reset the CPU affinity when we exit this script for any reason
  # This way either after the timer expires or after the process is interrupted
  # via ^C or SIGTERM, we return things back to the way they should be.
  trap setRestricted EXIT

  logger "Recovery: Recovery Mode Starting"
  setUnrestricted
  waitForReady
}

if [[ "${BASH_SOURCE[0]}" = "${0}" ]]; then
  main "${@}"
  exit $?
fi
