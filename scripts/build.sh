#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

source ${WORKINGDIR}/scripts/version.sh

mkdir -p $WORKINGDIR/bin
cd $WORKINGDIR/bin

echo "Start build Hangar binary version $VERSION"

BUILD_ARGS="-buildmode=pie"
BUILD_LDFAGS=""
BUILD_TAGS=""
if [[ "${DEBUG:-}" = "true" ]]; then
    echo "Debug enabled for the built binary file."
else
    echo "Build non-debug binary file with '-s -w' ldflags."
    BUILD_LDFAGS='-s -w'
fi

if [[ -n "${COMMIT}" ]]; then
    BUILD_LDFAGS="${BUILD_LDFAGS} -X 'github.com/cnrancher/hangar/pkg/utils.GitCommit=${COMMIT}'"
fi
BUILD_LDFAGS="${BUILD_LDFAGS} -X 'github.com/cnrancher/hangar/pkg/utils.Version=${VERSION}'"

if [[ -n "${DISABLE_CGO:-}" ]]; then
    export CGO_ENABLED=0
    BUILD_TAGS="containers_image_openpgp"
    BUILD_LDFAGS="${BUILD_LDFAGS} -extldflags='-static'"
    echo "CGO Disabled with '-static' extldflags."
fi

go build \
    "${BUILD_ARGS}" \
    -tags "${BUILD_TAGS}" \
    -ldflags "${BUILD_LDFAGS}" \
    -o hangar \
    "${WORKINGDIR}"

echo "--------------------------"
ls -alh hangar*
echo "--------------------------"
