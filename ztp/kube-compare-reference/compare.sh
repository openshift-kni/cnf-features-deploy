#! /bin/bash

DIFF=${DIFF:-colordiff}
if ! command -v "$DIFF" >/dev/null; then
    echo "Warning: Requested diff tool '$DIFF' is not found; falling back to plain old 'diff'"
    DIFF="diff"
fi

trap cleanup EXIT

function cleanup() {
  rm -rf source_file rendered_file same_file
}

function read_dir() {
  local dir=$1
  local file

  for file in $(ls "$dir"); do
    if [ -d "$dir""/""$file" ]; then
      read_dir "$dir""/""$file"
    else
      echo "$dir""/""$file"
    fi
  done
}

function compare_cr {
  local rendered_dir=$1
  local source_dir=$2
  local exclusionfile=$3
  local status=0

  read_dir "$rendered_dir" |grep yaml  > rendered_file
  read_dir "$source_dir" |grep yaml  > source_file

  local source_cr rendered
  while IFS= read -r source_cr; do
    while IFS= read -r rendered; do
      if [ "${source_cr##*/}" = "${rendered##*/}" ]; then
        # helm adds a yaml doc header (---) and a leading comment to every source_cr file; so remove those lines
        tail -n +3 "$rendered" > "$rendered.fixed"
        mv "$rendered.fixed" "$rendered"

        # Check the differences
        if ! $DIFF -u "$source_cr" "$rendered"; then
            status=$(( status || 1 ))
            printf "\n\n**********************************************************************************\n\n"
        fi
        # cleanup
        echo "$source_cr" >> same_file
      fi
    done < rendered_file
  done < source_file

  # Filter out files with a source-cr/reference match from the full list of potentiol source-crs/reference files
  while IFS= read -r file; do
    [[ ${file::1} != "#" ]] || continue # Skip any comment lines in the exclusionfile
    [[ -n ${file} ]] || continue # Skip empty lines
    sed -i "/${file##*/}/d" source_file
    sed -i "/${file##*/}/d" rendered_file
  done < <(cat same_file "$exclusionfile")

  if [[ -s source_file || -s rendered_file ]]; then
    [ -s source_file ] && printf "\n\nThe following files exist in source-crs only, but not found in reference:\n" && cat source_file
    [ -s rendered_file ] && printf "\nThe following files exist in reference only, but not found in source-crs:\n" && cat rendered_file
    status=1
  fi

  return $status
}

compare_cr renderedv1 ../source-crs compare_ignore
