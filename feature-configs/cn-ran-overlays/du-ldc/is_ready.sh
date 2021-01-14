#!/bin/sh
#
# Check DU-LDC deployment / configuration are complete

FLAVOR="-du-ldc"
ROLE="worker$FLAVOR"

####################################################
# Checks whether the machine config pool contains a 
#   specific module
# Globals:
#   ROLE - role suffix applied to node-role.kubernetes.io/
# Arguments:
#   MC module name to check for presense
# Outputs:
#   None
#   Aborts the script if the module is not present
####################################################
function is_mcp_ready ()
{
    MODULE=\"$1\"
    QUERY="{.spec.configuration.source[?(@.name==$MODULE)].name}"
    # If module name is not present in the MCP, abort (not ready)
    if [ -z "$(oc get mcp $ROLE -o jsonpath=$QUERY)" ]; then
        abort "$1 not picked"
    fi
}

####################################################
# Aborts script execution with a message if verbose 
# Globals:
#   VERBOSE - if defined, prints verbose status
# Arguments:
#   message
# Outputs:
#   None
#   Aborts the script
####################################################
function abort ()
{
    if [[ -n "${VERBOSE}" ]]; then
        echo "$1"
    fi
    exit 1
}

####################################################
# Checks that all configurations has been applied 
# Globals:
#   ROLE - role suffix applied to node-role.kubernetes.io/
#   FLAVOR -  suffix applied to "worker" role and all 
#       kubernetes object names specific to the flavor
# Arguments:
#   message
# Outputs:
#   None
#   Exits with code 0 if all applied, else code 1
####################################################
function main ()
{
    # Check machine config modules have been picked by MCO
    is_mcp_ready "load-sctp-module$FLAVOR"
    
    # TODO: Remove when integrated into PTP operator
    is_mcp_ready "disable-chronyd$FLAVOR"
    
    is_mcp_ready "performance-perf$FLAVOR"

    # Check kernel patch daemonset has been scheduled on 
    # all applicable nodes
    # TODO - remove when RT kernel patch is removed
    DS_MISS=$(oc -n default get ds/rtos$FLAVOR-ds -o \
        jsonpath='{.status.numberMisscheduled}')
    if [[ ${DS_MISS} -gt 0 ]]; then
        abort "Kernel patch daemonset is not updated yet"
    fi
    
    # TODO - remove when RT kernel patch is removed
    # Check hernel has been patched on all machines
    LST_KERNELS=$(oc get no -l node-role.kubernetes.io/$ROLE="" -o json \
        |grep '"kernelVersion": ')
    IFS=,
    for value in $LST_KERNELS;
    do
        if [[ -z $(echo $value |grep rt) ]]; then
        abort "Kernel has not been patched yet on all machines"
        fi
    done

    # Check MCP is updated
    oc wait mcp/$ROLE --for condition=updated --timeout 1s
}

main
