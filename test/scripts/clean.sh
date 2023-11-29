#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

source ./scripts/env.sh

files=(
    "*.zip"
    "*-failed.txt"
    ".pytest_cache"
    ".tox"
    "suite/converted.txt"
    "suite/*-failed.txt"
    "suite/*.zip"
    "suite/__pycache__"
)

for f in ${files[@]}; do
    if [[ -e "$f" ]]; then
        echo "Delete: $f"
        rm -rf $WORKINGDIR/$f
    fi
done

# Registry server.
if [[ $(docker ps -a -f "name=${HANGAR_REGISTRY_NAME}" --format=json) != "" ]]; then
    docker kill ${HANGAR_REGISTRY_NAME} > /dev/null || true
    docker rm ${HANGAR_REGISTRY_NAME} > /dev/null || true
    echo Delete ${HANGAR_REGISTRY_NAME}.
fi

exit 0
