#! /bin/bash
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
  status=0

  read_dir "$rendered_dir" |grep yaml  > rendered_file
  read_dir "$source_dir" |grep yaml  > source_file


  while IFS= read -r file1; do
    while IFS= read -r file2; do
      if [ "${file1##*/}" = "${file2##*/}" ]; then
        diff -u "$file1" "$file2"
        status=$(( "$status" || $? ))
        printf "\n\n" 
        echo "$file1" >> same_file
      fi
    done < rendered_file
  done < source_file


  while IFS= read -r file; do
    sed -i "/${file##*/}/d" source_file
    sed -i "/${file##*/}/d" rendered_file
  done < same_file

}



compare_cr renderedv1 ../source-crs

if [[ -s source_file || -s rendered_file ]]; then
  [ -s source_file ] && printf "\n\nThe following files exist in source-crs only, but not found in reference:\n" && cat source_file
  [ -s rendered_file ] && printf "\nThe following files exist in reference only, but not found in source-crs:\n" && cat rendered_file
  exit 1
else
  exit $status
fi
