#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

source ${WORKINGDIR}/scripts/version.sh

mkdir -p $WORKINGDIR/bin
cd $WORKINGDIR/bin

echo "Start build Hangar binary..."

BUILD_ARGS="-buildmode=pie"
BUILD_LDFAGS=""
BUILD_TAGS=""
if [[ -z "${DEBUG:-}" ]]; then
    BUILD_LDFAGS="-extldflags -static -s -w"
else
    echo "Debug enabled for the built binary file."
fi
if [[ ! -z "${COMMIT}" ]]; then
    BUILD_LDFAGS="${BUILD_LDFAGS} -X 'github.com/cnrancher/hangar/pkg/utils.GitCommit=${COMMIT}'"
fi
BUILD_LDFAGS="${BUILD_LDFAGS} -X 'github.com/cnrancher/hangar/pkg/utils.Version=${VERSION}'"

if [[ ! -z ${DISABLE_CGO:-} ]]; then
    export CGO_ENABLED=0
    BUILD_TAGS="containers_image_openpgp"
fi

go build \
    "${BUILD_ARGS}" \
    -tags "${BUILD_TAGS}" \
    -ldflags "${BUILD_LDFAGS}" \
    -o hangar \
    ${WORKINGDIR}

echo "--------------------------"
ls -alh hangar*
echo "--------------------------"
