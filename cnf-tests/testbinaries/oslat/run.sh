#!/usr/bin/env bash

set -euo pipefail

# env vars:
# RTPRIO (default to 1): set the SCHED_FIFO priority
# RUNTIME_SECONDS (default 10 minute): how long the test will run
# LOG_DIR (default stdout): the log directory where to save the ouput of the oslat command

RTPRIO=${RTPRIO:-1}
RUNTIME_SECONDS=${RUNTIME_SECONDS:-600}

cpulist=()
for cpu in $(cat /proc/self/status | grep Cpus_allowed_list: | cut -f 2 | awk '/-/{for (i=$1; i<=$2; i++)printf "%s%s",i,ORS;next} 1' RS=, FS=-)
do
	cpulist+=(${cpu})
done

echo "############# dumping env ###########"
env
echo "#####################################"

echo " "
echo "########## container info ###########"
echo "/proc/cmdline: $(cat /proc/cmdline)"
echo "allowed cpu list: ${cpulist[@]}"
echo "uname -nr: $(uname -nr)"
echo "#####################################"

main_thread_cpu="${cpulist[0]}"
main_thread_cpu_sibling=$(cat /sys/devices/system/cpu/cpu${main_thread_cpu}/topology/thread_siblings_list | awk -F '[-,]' '{print $2}')

cyccore="${cpulist[1]}"
for cpu in "${cpulist[@]:2}"
do
	if [[ "${cpu}" == "${main_thread_cpu_sibling}" ]]; then
		continue
	fi
	cyccore="${cyccore},${cpu}"
done

if [[ "${cyccore}" == "" ]]; then
	exit 1
fi

log_out="/dev/stderr"
if [[ "${LOG_DIR}" != "" ]]; then
  log_out="${LOG_DIR}/oslat.log"
fi

# run the oslat command in the separate process
echo "cmd to run: oslat --runtime ${RUNTIME_SECONDS} --rtprio ${RTPRIO} --cpu-list ${cyccore} --cpu-main-thread ${main_thread_cpu}"
/usr/bin/oslat --runtime "${RUNTIME_SECONDS}" --rtprio "${RTPRIO}" --cpu-list "${cyccore}" --cpu-main-thread "${main_thread_cpu}" > "${log_out}"

echo "DONE"
