#!/bin/bash

source $(dirname "$0")/common.sh
init $1

python watcher.py $(./get-pre-sync-rv.sh $1) $RESOURCE_NAME debug
