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

KUBELET_CPU_STATE=/var/lib/kubelet/cpu_manager_state
FULL_CPU_STATE=/sys/fs/cgroup/cpuset/cpuset.cpus
unrestrictedCpuset() {
  local cpus
  if [[ -e $KUBELET_CPU_STATE ]]; then
      cpus=$(jq -r '.defaultCpuSet' <$KUBELET_CPU_STATE)
  fi
  if [[ -z $cpus ]]; then
    # fall back to using all cpus if the kubelet state is not configured yet
    [[ -e $FULL_CPU_STATE ]] || return 1
    cpus=$(<$FULL_CPU_STATE)
  fi
  echo $cpus
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

getCPUCount () {
  local cpuset="$1"
  local cpulist=()
  local cpus=0
  local mincpus=2

  if [[ -z $cpuset || $cpuset =~ [^0-9,-] ]]; then
    echo $mincpus
    return 1
  fi

  IFS=',' read -ra cpulist <<< $cpuset

  for elm in "${cpulist[@]}"; do
    if [[ $elm =~ ^[0-9]+$ ]]; then
      (( cpus++ ))
    elif [[ $elm =~ ^[0-9]+-[0-9]+$ ]]; then
      local low=0 high=0
      IFS='-' read low high <<< $elm
      (( cpus += high - low + 1 ))
    else
      echo $mincpus
      return 1
    fi
  done

  # Return a minimum of 2 cpus
  echo $(( cpus > $mincpus ? cpus : $mincpus ))
  return 0
}

resetOVSthreads () {
  local cpucount="$1"
  local curRevalidators=0
  local curHandlers=0
  local desiredRevalidators=0
  local desiredHandlers=0
  local rc=0

  curRevalidators=$(ps -Teo pid,tid,comm,cmd | grep -e revalidator | grep -c ovs-vswitchd)
  curHandlers=$(ps -Teo pid,tid,comm,cmd | grep -e handler | grep -c ovs-vswitchd)

  # Calculate the desired number of threads the same way OVS does.
  # OVS will set these thread count as a one shot process on startup, so we
  # have to adjust up or down during the boot up process. The desired outcome is
  # to not restrict the number of thread at startup until we reach a steady
  # state.  At which point we need to reset these based on our restricted  set
  # of cores.
  # See OVS function that calculates these thread counts:
  # https://github.com/openvswitch/ovs/blob/master/ofproto/ofproto-dpif-upcall.c#L635
  (( desiredRevalidators=$cpucount / 4 + 1 ))
  (( desiredHandlers=$cpucount - $desiredRevalidators ))


  if [[ $curRevalidators -ne $desiredRevalidators || $curHandlers -ne $desiredHandlers ]]; then

    logger "Recovery: Re-setting OVS revalidator threads: ${curRevalidators} -> ${desiredRevalidators}"
    logger "Recovery: Re-setting OVS handler threads: ${curHandlers} -> ${desiredHandlers}"

    ovs-vsctl set \
      Open_vSwitch . \
      other-config:n-handler-threads=${desiredHandlers} \
      other-config:n-revalidator-threads=${desiredRevalidators}
    rc=$?
  fi

  return $rc
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

  resetOVSthreads "$(getCPUCount ${cpuset})"
  if [[ $? -ne 0 ]]; then
    ((failcount++))
  else
    ((successcount++))
  fi

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
