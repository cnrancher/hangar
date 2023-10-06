#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

source ${WORKINGDIR}/scripts/version.sh

mkdir -p $WORKINGDIR/bin
cd $WORKINGDIR/bin

BUILD_FLAG=""
if [[ -z "${DEBUG:-}" ]]; then
    BUILD_FLAG="-extldflags -static -s -w"
else
    echo "Debug enabled"
fi
if [[ ! -z "${COMMIT}" ]]; then
    BUILD_FLAG="${BUILD_FLAG} -X 'github.com/cnrancher/hangar/pkg/utils.GitCommit=${COMMIT}'"
fi
BUILD_FLAG="${BUILD_FLAG} -X 'github.com/cnrancher/hangar/pkg/utils.Version=${VERSION}'"

CGO_ENABLED=0 GOOS=$OS GOARCH=$ARCH go build -ldflags "${BUILD_FLAG}" -o hangar ..

echo "--------------------------"
ls -alh hangar*
echo "--------------------------"
