#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

source ${WORKINGDIR}/scripts/env.sh

RUNNER_ARCH=$(uname -m)
case ${RUNNER_ARCH} in
    amd64 | x86_64)
        ARCH="amd64"
        ;;
    arm64 | aarch64)
        ARCH="arm64"
        ;;
    *)
        echo "Unrecognized arch: ${RUNNER_ARCH}"
        exit 1
        ;;
esac

RUNNER_OS=$(uname -s)
case ${RUNNER_OS} in
    Darwin)
        OS="darwin"
        ;;
    Linux)
        OS="linux"
        ;;
    *)
        echo "Unrecognized OS: ${RUNNER_OS}"
        exit 1
        ;;
esac


mkdir -p $WORKINGDIR/build
cd $WORKINGDIR/build

OUTPUT="hangar-$OS-$ARCH-$VERSION"
CGO_ENABLED=0 GOOS=$OS GOARCH=$ARCH go build -ldflags "${BUILD_FLAG}" -o $OUTPUT ..
echo $(pwd)/$OUTPUT
