#!/usr/bin/env bash

set -e

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

TAG=${TAG:="image-tools"}
VERSION=${VERSION:=$(git describe --tags 2>/dev/null || echo "")}
REGISTRY=${REGISTRY:="docker.io/cnrancher"}
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

docker build --tag "${REGISTRY}/${TAG}:${VERSION}-amd64" \
    --build-arg SKOPEO_DIGEST="sha256:508176cb1a969c08265fe7c48e7223a9e236fc5e2851f40bfc21ae7c73af4249" \
    --build-arg ARCH="amd64" \
    --platform linux/amd64 \
    -f Dockerfile .

docker build --tag "${REGISTRY}/${TAG}:${VERSION}-arm64" \
    --build-arg SKOPEO_DIGEST="sha256:ab1c8fd678183a390df15aebc5c16d0c541cd61342602c6b1a16d9bc38fa9c57" \
    --build-arg ARCH="arm64" \
    --platform linux/arm64 \
    -f Dockerfile .

docker push "${REGISTRY}/${TAG}:${VERSION}-amd64"
docker push "${REGISTRY}/${TAG}:${VERSION}-arm64"

docker buildx imagetools create --tag "${REGISTRY}/${TAG}:${VERSION}" \
    "${REGISTRY}/${TAG}:${VERSION}-arm64" \
    "${REGISTRY}/${TAG}:${VERSION}-amd64"
