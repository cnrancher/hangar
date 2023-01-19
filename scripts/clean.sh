#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

files=(
    "image-tools"
    "build/"
    "pkg/archive/part/test/test*"
    "pkg/rancher/chart/test/*"
)

for f in ${files[@]}; do
    if [[ -e "$f" ]]; then
        echo "Delete: $f"
        rm -r $WORKINGDIR/$f &> /dev/null
    fi
done

exit 0
