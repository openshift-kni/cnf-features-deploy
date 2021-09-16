#!/bin/bash

. $(dirname $0)/common.sh
./hack/build.sh

XDG_CONFIG_HOME=./ $(getKustomize) build --enable-alpha-plugins
