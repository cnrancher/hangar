#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

source scripts/env.sh
mkdir -p $WORKINGDIR/registry

if [[ $(uname -s) = "Darwin" ]]; then
    echo "The registry port '5000' is conflict with the macOS system service."
    echo "Use other OS to run tests or use another registry server by specify"
    echo "REGISTRY_URL env manually."
    exit 1
fi

if [[ $(docker ps -a -f "name=${HANGAR_REGISTRY_NAME}" --format=json) != "" ]]; then
    docker kill ${HANGAR_REGISTRY_NAME} > /dev/null || true
    docker rm ${HANGAR_REGISTRY_NAME} > /dev/null || true
    echo Delete ${HANGAR_REGISTRY_NAME}.
fi

echo "Starting ${HANGAR_REGISTRY_NAME}"
docker run -d --rm \
    -p ${HANGAR_REGISTRY_PORT}:5000 \
    -v $WORKINGDIR/registry:/var/lib/registry \
    --name ${HANGAR_REGISTRY_NAME} \
    registry:2
