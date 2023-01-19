#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

source ${WORKINGDIR}/scripts/env.sh

ARCH=('amd64' 'arm64')
OS=('linux' 'darwin')

mkdir -p $WORKINGDIR/build
cd $WORKINGDIR/build

for os in ${OS[@]}
do
    for arch in ${ARCH[@]}
    do
        OUTPUT="image-tools-$os-$arch-$VERSION"
        GOOS=$os GOARCH=$arch go build -ldflags "${BUILD_FLAG}" -o $OUTPUT ..
        echo $(pwd)/$OUTPUT
    done
done
