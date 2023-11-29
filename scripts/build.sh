#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

source ${WORKINGDIR}/scripts/version.sh

mkdir -p $WORKINGDIR/bin
cd $WORKINGDIR/bin

echo "Start build Hangar binary..."

BUILD_FLAG=""
if [[ -z "${DEBUG:-}" ]]; then
    BUILD_FLAG="-extldflags -static -s -w"
else
    echo "Debug enabled for the built binary file."
fi
if [[ ! -z "${COMMIT}" ]]; then
    BUILD_FLAG="${BUILD_FLAG} -X 'github.com/cnrancher/hangar/pkg/utils.GitCommit=${COMMIT}'"
fi
BUILD_FLAG="${BUILD_FLAG} -X 'github.com/cnrancher/hangar/pkg/utils.Version=${VERSION}'"

if [[ ! -z ${DISABLE_CGO:-} ]]; then
    CGO_ENABLED=0 \
        GOOS=$OS \
        GOARCH=$ARCH \
        go build \
        -tags containers_image_openpgp \
        -ldflags "${BUILD_FLAG}" -o hangar ..
else
    GOOS=$OS \
        GOARCH=$ARCH \
        go build -ldflags "${BUILD_FLAG}" -o hangar ..
fi

echo "--------------------------"
ls -alh hangar*
echo "--------------------------"
