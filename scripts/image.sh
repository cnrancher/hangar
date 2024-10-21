#!/bin/bash

set -euo pipefail

cd $(dirname $0)/../

REPO=${REPO:-'cnrancher'}
TAG=${TAG:-'latest'}
BUILDER='hangar-builder'
TARGET_PLATFORMS='linux/arm64,linux/amd64'

# FYI: https://docs.docker.com/build/buildkit/toml-configuration/#buildkitdtoml
BUILDX_CONFIG_DIR=${BUILDX_CONFIG_DIR:-"$HOME/.config/buildkit/"}
BUILDX_CONFIG=${BUILDX_CONFIG:-"$HOME/.config/buildkit/buildkitd.toml"}
BUILDX_OPTIONS=${BUILDX_OPTIONS:-''} # Set to '--push' to upload images

if [[ ! -e "${BUILDX_CONFIG}" ]]; then
    mkdir -p ${BUILDX_CONFIG_DIR}
    touch ${BUILDX_CONFIG}
fi

docker buildx ls | grep ${BUILDER} || \
    docker buildx create \
        --config ${BUILDX_CONFIG} \
        --driver-opt network=host \
        --name=${BUILDER} \
        --platform=${TARGET_PLATFORMS}

echo "Start build images"
set -x

docker buildx build -f package/Dockerfile \
    --builder ${BUILDER} \
    -t "${REPO}/hangar:${TAG}" \
    --platform=${TARGET_PLATFORMS} ${BUILDX_OPTIONS} .

set +x
echo "Image: Done"
