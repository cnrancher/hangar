#!/usr/bin/env bash

set -e

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

if [[ -z "${DOCKER_USERNAME}" || -z "${DOCKER_PASSWORD}" ]]; then
    echo "DOCKER_USERNAME or DOCKER_PASSWORD not set"
    exit 1
fi

echo "${DOCKER_PASSWORD}" | docker login \
    --username ${DOCKER_USERNAME} \
    --password-stdin

export TAG=${TAG:="image-tools"}
export VERSION=${VERSION:=$(git describe --tags 2>/dev/null || echo "")}
export REGISTRY=${REGISTRY:-"docker.io/cnrancher"}
if [[ "${VERSION}" = "" ]]; then
    if [[ "${DRONE_TAG}" != "" ]]; then
        echo "DRONE_TAG: ${DRONE_TAG}"
        VERSION=${DRONE_TAG}
    else
        echo "DRONE_COMMIT_SHA: ${DRONE_COMMIT_SHA}"
        VERSION=${DRONE_COMMIT_SHA:0:8}
    fi
fi
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
