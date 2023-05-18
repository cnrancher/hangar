#!/bin/bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

docker build --tag "${REGISTRY}/${TAG}:${VERSION}-arm64" \
    --build-arg ARCH="arm64" \
    --platform linux/arm64 \
    -f package/Dockerfile .

docker push "${REGISTRY}/${TAG}:${VERSION}-arm64"
