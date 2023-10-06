#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

files=(
    ".dapper*"
    "hangar"
    "bin/"
    "dists/"
    "pkg/archive/part/test/test*"
    "pkg/rancher/chartimages/test/*"
    "pkg/rancher/kdmimages/test/*"
)

for f in ${files[@]}; do
    if [[ -e "$f" ]]; then
        echo "Delete: $f"
        rm -rf $WORKINGDIR/$f
    fi
done

exit 0
