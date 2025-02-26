#!/bin/bash

# check if any filename matches a skipped file pattern
check_skipped_files() {
    local filenames="$1"
    local is_skipped_file=false

    # source the skipped files config
    . "$(dirname "${BASH_SOURCE[0]}")/skipped-files.sh"

    # check if we must skip any check
    for filename in $filenames; do
        for skipped_file in "${skipped_files[@]}"; do
            if [[ "$filename" =~ $skipped_file ]]; then
                is_skipped_file=true
                break
            else
                is_skipped_file=false
            fi
        done
        if [ "$is_skipped_file" = false ]; then
            break
        fi
    done

    if [ "$is_skipped_file" = true ]; then
        echo "INFO: skipping commit msg check"
        return 0
    fi
    return 1  # return non-zero if no files are skipped
}
