#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

files=(
    "load-part.*"
    "load-directory"
    "*-failed.txt"
    "*.tar.gz"
    "*.tar.zstd"
    "*.txt"
    "saved-images"
    "saved-image-cache"
    "charts-repo-cache"
    "pandaria-catalog"
    "rancher-charts"
    "system-charts"
    "data.json"
    "*.part*"
)

for f in ${files[@]}; do
    if [[ -e "$f" ]]; then
        echo "Delete: $f"
        rm -rf $WORKINGDIR/$f
    fi
done

exit 0
