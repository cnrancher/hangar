#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

files=(
    "*.zip"
    "*-failed.txt"
    "*.key"
    "*.pub"
    ".pytest_cache"
    ".tox"
    "suite/converted.txt"
    "suite/*-failed.txt"
    "suite/*.zip"
    "suite/*.key"
    "suite/*.pub"
    "suite/scan-report.*"
    "suite/*.csv"
    "suite/__pycache__"
    "suite/.pytest_cache"
)

for f in ${files[@]}; do
    if [[ -e "$f" ]]; then
        echo "Delete: $f"
        rm -rf $WORKINGDIR/$f
    fi
done

exit 0
