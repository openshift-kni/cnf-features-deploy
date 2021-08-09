#!/bin/bash

. $(dirname "$0")/common.sh $1

python watcher.py $(./get-pre-sync-rv.sh) $RESOURCE_NAME debug