#!/usr/bin/env bash

set -euo pipefail

# Launch HTTP distribution registry server in docker
# URL: http://127.0.0.1:5000
function setup_distribution_registry() {
    set -euo pipefail

    HANGAR_REGISTRY_NAME=hangar-registry
    HANGAR_REGISTRY_PORT=5000
    WORKINGDIR=${WORKINGDIR}

    type docker > /dev/null

    mkdir -p $WORKINGDIR/registry
    if [[ $(docker ps -a -f "name=${HANGAR_REGISTRY_NAME}" --format=json) != "" ]]; then
        docker kill ${HANGAR_REGISTRY_NAME} > /dev/null || true
        docker rm ${HANGAR_REGISTRY_NAME} > /dev/null || true
        echo Delete ${HANGAR_REGISTRY_NAME} docker.
    fi

    echo "Starting ${HANGAR_REGISTRY_NAME}"
    docker run -d --rm \
        -p ${HANGAR_REGISTRY_PORT}:${HANGAR_REGISTRY_PORT} \
        -v $WORKINGDIR/registry:/var/lib/registry \
        --name ${HANGAR_REGISTRY_NAME} \
        registry:2
}

function delete_distribution_registry() {
    set -euo pipefail

    HANGAR_REGISTRY_NAME=${HANGAR_REGISTRY_NAME}

    docker kill ${HANGAR_REGISTRY_NAME} > /dev/null || true
    docker rm ${HANGAR_REGISTRY_NAME} > /dev/null || true
    echo Delete ${HANGAR_REGISTRY_NAME} docker.
}
