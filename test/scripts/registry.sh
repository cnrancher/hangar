#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

source scripts/env.sh
mkdir -p $WORKINGDIR/registry

if [[ $(docker ps -a -f "name=${HANGAR_REGISTRY_NAME}" --format=json) != "" ]]; then
    docker kill ${HANGAR_REGISTRY_NAME} > /dev/null || true
    docker rm ${HANGAR_REGISTRY_NAME} > /dev/null || true
    echo Delete ${HANGAR_REGISTRY_NAME}.
fi

echo "Starting ${HANGAR_REGISTRY_NAME}"
docker run -d \
	--restart=always \
    -p ${HANGAR_REGISTRY_PORT}:5000 \
    -v $WORKINGDIR/registry:/var/lib/registry \
    --name ${HANGAR_REGISTRY_NAME} \
    registry:2
