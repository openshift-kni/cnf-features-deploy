#!/bin/bash
#
# Generate symlinks to mimic the older directory structure in 4.19 to allow customers time to adapt to the new layout
#
# Note: Invoke with DRY_RUN=true in the environment for testing
#

set -euo pipefail

if [[ $# -lt 2 || $1 == "--help" || $1 == "-h" ]]; then
  echo "Usage:"
  echo "  ${0##*/} path/to/manifest path/to/source-crs"
  exit 1
fi

MANIFEST_PATH=$1
SOURCE_CRS_PATH=$2

readarray -t legacy_paths <"$MANIFEST_PATH"

cd "$SOURCE_CRS_PATH"

echo "Generating legacy source-cr paths from $MANIFEST_PATH:"
ERRORS=0
for legacy_path in "${legacy_paths[@]}"; do
  if [[ -z $legacy_path ]]; then
    continue
  fi

  echo -n "  $legacy_path: "

  if [[ -e $legacy_path ]]; then
    echo "exists"
    continue
  fi

  readarray -t new_location_matches < <(find . -name "${legacy_path##*/}")

  if [[ ${#new_location_matches[@]} -eq 0 ]]; then
    echo "NO MATCH"
    ((ERRORS++))
    continue
  fi

  if [[ ${#new_location_matches[@]} -eq 1 ]]; then
    # Exactly one match: use it
    new_location=${new_location_matches[0]}
  else
    # In case this is an architecture-specific duplication, always choose the one with x86 in the name
    filtered_list=()
    for location in "${new_location_matches[@]}"; do
      if [[ $location =~ x86 ]]; then
        filtered_list+=("$location")
      fi
    done
    if [[ ${#filtered_list[@]} -ne 1 ]]; then
      echo "MULTIPLE MATCHES:"
      for location in "${new_location_matches[@]}"; do
        echo "  $location"
      done
      ((ERRORS++))
      continue
    fi
    new_location=${filtered_list[0]}
  fi

  # Strip leading path that was added by 'find'
  new_location=${new_location#./}

  # In case legacy_path is in a subdirectory, manufacture a relative path:
  if legacy_path_depth=$(grep -o '/' <<<"$legacy_path" | wc -l); then
    for ((counter = 0; counter < legacy_path_depth; counter++)); do
      new_location="../$new_location"
    done
  fi

  if [[ -n ${DRY_RUN:-} ]]; then
    echo "symlink to $new_location (dry run)"
  else
    mkdir -p "$(dirname "$legacy_path")"
    ln -s "$new_location" "$legacy_path"
    echo "symlinked to $new_location"
  fi
done

[[ $ERRORS -eq 0 ]]

