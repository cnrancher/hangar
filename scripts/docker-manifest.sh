#!/bin/bash

set -e

if [[ -z "${DOCKER_USERNAME}" || -z "${DOCKER_PASSWORD}" ]]; then
    echo "DOCKER_USERNAME or DOCKER_PASSWORD not set"
    exit 1
fi

# Check docker buildx installed or not
if ! docker buildx version &> /dev/null ; then
    BUILDX_ARCH=""
    case $(uname -m) in
        x86_64)
            BUILDX_ARCH="amd64"
            ;;
        aarch64)
            BUILDX_ARCH="arm64"
            ;;
        *)
            echo "unrecognized arch: $(uname -m)"
            echo "Please install docker-buildx manually"
            exit 1
            ;;
    esac
    echo "docker buildx arch: $BUILDX_ARCH"
    # Add buildx plugin from github
    mkdir -p ${HOME}/.docker/cli-plugins/ && \
    curl -sLo ${HOME}/.docker/cli-plugins/docker-buildx \
        https://github.com/docker/buildx/releases/download/v0.9.1/buildx-v0.9.1.linux-${BUILDX_ARCH} && \
    chmod +x ${HOME}/.docker/cli-plugins/docker-buildx
fi

echo "${DOCKER_PASSWORD}" | docker login \
    --username ${DOCKER_USERNAME} \
    --password-stdin

export TAG=${TAG:-"image-tools"}
export VERSION=${VERSION:-$(git describe --tags 2>/dev/null || echo "")}
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

docker buildx imagetools create --tag "${REGISTRY}/${TAG}:${VERSION}" \
    "${REGISTRY}/${TAG}:${VERSION}-arm64" \
    "${REGISTRY}/${TAG}:${VERSION}-amd64"

# update latest tag
if [[ "${SKIP_LATEST_TAG}" != "1" ]]; then
    docker buildx imagetools create --tag "${REGISTRY}/${TAG}:latest" \
        "${REGISTRY}/${TAG}:${VERSION}-arm64" \
        "${REGISTRY}/${TAG}:${VERSION}-amd64"
fi
