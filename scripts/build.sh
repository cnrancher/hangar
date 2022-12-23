#!/usr/bin/env bash

set -e

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

ARCH=('amd64' 'arm64')
OS=('linux' 'darwin')
VERSION=$(git describe --tags)

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
