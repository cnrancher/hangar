#!/bin/bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR="$(pwd)"

source scripts/env.sh

# Set-up the registry server
${WORKINGDIR}/scripts/registry.sh

export REGISTRY_IP=$(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' ${HANGAR_REGISTRY_NAME})
export REGISTRY_URL="${REGISTRY_IP}:${HANGAR_REGISTRY_PORT}"

echo "REGISTRY_URL: ${REGISTRY_URL}"

tox $@

if [[ $(docker ps -a -f "name=${HANGAR_REGISTRY_NAME}" --format=json) != "" ]]; then
    docker kill ${HANGAR_REGISTRY_NAME} > /dev/null || true
    docker rm ${HANGAR_REGISTRY_NAME} > /dev/null || true
    echo Delete ${HANGAR_REGISTRY_NAME}.
fi
