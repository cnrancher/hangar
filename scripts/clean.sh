#!/bin/bash

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

files=(
    "image-tools"
    "mirror-failed.txt"
    "load-failed.txt"
    "save-failed.txt"
    "output/"
)

for f in ${files[@]}; do
    echo "Delete: $f"
    rm -r $WORKINGDIR/$f &> /dev/null
done

exit 0
