#!/bin/sh
#
# Check DU-LDC deployment / configuration are complete

FLAVOR="-cu-up"
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
    
    is_mcp_ready "performance-perf$FLAVOR"

    # Check MCP is updated
    oc wait mcp/$ROLE --for condition=updated --timeout 1s
}

main
