#!/bin/bash

oc wait ${CNF_MCP:-"mcp/worker-cnf"} --for condition=updated --timeout 1s
