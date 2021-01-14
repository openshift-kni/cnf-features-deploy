#!/bin/sh
#
# Clone one example to another

if [[ -z $1 || -z $2 || -z $3 ]]; then
  echo "Usage:"
  echo "./clone.sh <path> <from> <to>"
  echo "<path> - path to the <from> or the <to>,\
  for example '..'"
  echo "<from> - the name of the source forder,\
  for example 'du-ldc'"
  echo "<to> - the name of the destination forder,\
  for example du-ldc-2"
  echo 'For example: "./clone.sh .. du-ldc du-ldc-2"'
  exit 1
fi

if [[ ! -d $1/$2 ]]; then
  echo "ERROR: The source path provided does not exest"
  exit 1
fi

# Clone the source directory under the new name
cp -rf $1/$2 $1/$3

# Rename all files that contain the old flavor in their name
RN_FILES=$(find $1/$3 -type f -name *$2*)
for f in $RN_FILES; do
  FN=$(basename $f)
  NEW_FN=$(echo $FN | sed -e s/$2/$3/)
  mv $f $(dirname $f)/$NEW_FN
done

# Network resource can't contain dashes
# We replace them by underscores
NR_FROM=$(echo $2 | sed -e 's/-/_/')
NR_TO=$(echo $3 | sed -e 's/-/_/')
find $1/$3 -type f -exec sed -i -e s/$2/$3/g {} +
find $1/$3 -type f -exec sed -i -e s/$NR_FROM/$NR_TO/g {} +
echo "Done"

