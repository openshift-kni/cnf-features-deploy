#!/bin/sh

set -eu

proxy=${1:?"usage: $0 QUAY_PROXY < Dockerfile"}

while IFS= read -r line || [ -n "$line" ]; do
    case "$line" in
        "FROM registry.ci.openshift.org/"*)
            image_and_rest=${line#FROM }
            image=${image_and_rest%% *}
            rest=${image_and_rest#"$image"}
            source_ref=${image#registry.ci.openshift.org/}
            repository=${source_ref%:*}
            tag=${source_ref##*:}
            proxy_repository=$(printf '%s' "$repository" | tr '/' '_')

            printf 'FROM %s:%s_%s%s\n' \
                "$proxy" "$proxy_repository" "$tag" "$rest"
            ;;
        *)
            printf '%s\n' "$line"
            ;;
    esac
done
