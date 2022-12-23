#!/usr/bin/env bash

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

files=(
    "image-tools"
    "build/"
)

for f in ${files[@]}; do
    if [ -e "$f" ]; then
        echo "Delete: $f"
        rm -r $WORKINGDIR/$f &> /dev/null
    fi
done

exit 0
