#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

if [[ -z "${DOCKER_USERNAME}" || -z "${DOCKER_PASSWORD}" ]]; then
    echo "DOCKER_USERNAME or DOCKER_PASSWORD not set"
    exit 1
fi

echo "${DOCKER_PASSWORD}" | docker login \
    --username ${DOCKER_USERNAME} \
    --password-stdin

source ${WORKINGDIR}/scripts/env.sh

export TAG=${TAG:="hangar"}
export REGISTRY=${REGISTRY:-"docker.io/cnrancher"}
export VERSION=${VERSION}
echo "version: ${VERSION}"
echo "TAG: ${TAG}:${VERSION}"

RUNNER_ARCH=$(uname -m)
case ${RUNNER_ARCH} in
    amd64 | x86_64)
        ${WORKINGDIR}/scripts/docker-amd64.sh
        ;;
    arm64 | aarch64)
        ${WORKINGDIR}/scripts/docker-arm64.sh
        ;;
    *)
        echo "Unrecognized arch: ${RUNNER_ARCH}"
        ;;
esac
