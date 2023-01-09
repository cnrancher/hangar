#!/bin/bash

set -e

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

docker build --tag "${REGISTRY}/${TAG}:${VERSION}-arm64" \
    --build-arg ARCH="arm64" \
    --platform linux/arm64 \
    -f Dockerfile .

docker push "${REGISTRY}/${TAG}:${VERSION}-arm64"
