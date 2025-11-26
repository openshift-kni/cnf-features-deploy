#!/bin/bash
# Purpose: This script ensures that specific files containing hardcoded release versions are up to date with the current release and do not reference any previous versions.

# Functionality:
# 1. Determines if the current branch is a release branch (e.g., release-4.x).
# 2. Extracts the previous version (e.g. 4.16 for release-4.17).
# 3. Checks specific files for any occurrences of the previous version.
# 4. If matches are found, it prompts the user to update references to the current release branch and exits with an error code.
# 5. Skips checks for non-release branches or missing files.

set -x

branch_name=$(git rev-parse --abbrev-ref HEAD)

release_regex="release-([0-9]+)\.([0-9]+)"

if [[ ! $branch_name =~ $release_regex ]]; then
    echo "Branch '$branch_name' is not a release branch. Skipping checks."
    exit 0
fi

x="${BASH_REMATCH[1]}"
y="${BASH_REMATCH[2]}"
previous_version="$x.$((y - 1))" # Example: 4.16 for branch release-4.17

files_to_check=(
    "cnf-tests/Dockerfile.openshift"
    "cnf-tests/.konflux/Dockerfile"
    "cnf-tests/mirror/images.json"
    "cnf-tests/testsuites/pkg/images/images.go"
    "hack/common.sh"
    "hack/run-functests.sh"
)

match_found=false

for file in "${files_to_check[@]}"; do
    if [[ ! -f "$file"  ]]; then
        echo "Warning: File '$file' does not exist. Skipping."
        continue
    fi
    if grep -nw "$file" -e "$previous_version" 2>/dev/null ; then
        echo "Reference to $previous_version found in $file."
        match_found=true
    fi
done

if $match_found; then
    echo "----------------------------------------------------------------------------------"
    echo "The files above contain references to the previous release version ($previous_version)."
    echo "Please update them to the current release branch ($branch_name) to ensure consistency."
    echo "This can be done by creating a PR against ($branch_name)."
    echo "For more details, refer to the Branching section in the project's readme.md."
    exit 1
else
    echo "All files are up-to-date. No references to $previous_version were found."
fi
