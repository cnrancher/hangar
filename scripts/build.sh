#!/usr/bin/env bash

set -e

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

ARCH=('amd64' 'arm64')
OS=('linux' 'darwin')
VERSION=$(git describe --tags 2>/dev/null || echo "")
if [[ "${VERSION}" = "" ]]; then
    if [[ -n "${DRONE_TAG}" ]]; then
        echo "DRONE_TAG: ${DRONE_TAG}"
        VERSION=${DRONE_TAG}
    elif [[ -n "${DRONE_COMMIT_SHA}" ]]; then
        echo "DRONE_COMMIT_SHA: ${DRONE_COMMIT_SHA}"
        VERSION=${DRONE_COMMIT_SHA:0:8}
    else
        VERSION=$(git rev-parse --short HEAD)
    fi
fi
echo "Build version: ${VERSION}"

mkdir -p $WORKINGDIR/build
cd $WORKINGDIR/build

for os in ${OS[@]}
do
    for arch in ${ARCH[@]}
    do
        OUTPUT="image-tools-$os-$arch-$VERSION"
        GOOS=$os GOARCH=$arch go build -ldflags "-s -w" -o $OUTPUT ..
        echo $(pwd)/$OUTPUT
    done
done
