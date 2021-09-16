#!/bin/bash

set -e

pushd "$(dirname "$0")/.." >&2

function finish {
    popd >&2
}
trap finish EXIT

GOPATH="${GOPATH:-~/go}"
export GOFLAGS="${GOFLAGS:-"-mod=vendor"}"

export PATH=$PATH:$GOPATH/bin

PLUGINPATH=kustomize/plugin/policyGenerator/v1/policygenerator/
PLUGINBIN=PolicyGenerator

getKustomize() {
    local kustomize_version=$1
    local kustomize_dir=/tmp/policygenKustomize
    kustomize=$kustomize_dir/kustomize
    if [[ -x $kustomize ]]; then
        echo "Found cached kustomize at $kustomize" >&2
    else
        echo "Installing kustomize $kustomize_version into $kustomize_dir" >&2
        [[ -d $kustomize_dir ]] || mkdir -p $kustomize_dir >&2
        pushd $kustomize_dir >&2
        set +e
        curl -m 600 -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" \
          | bash -s $kustomize_version >&2
        set -e
        popd >&2
    fi
    # Log the version of kustomize we found
    $kustomize version >&2
    echo $kustomize
}
