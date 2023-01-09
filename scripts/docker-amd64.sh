#!/bin/bash

set -e

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

docker build --tag "${REGISTRY}/${TAG}:${VERSION}-amd64" \
    --build-arg ARCH="amd64" \
    --platform linux/amd64 \
    -f Dockerfile .

docker push "${REGISTRY}/${TAG}:${VERSION}-amd64"
