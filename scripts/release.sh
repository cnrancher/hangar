#!/bin/bash

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

ARCH=('amd64' 'arm64')
OS=('linux' 'darwin')

if [[ -z $VERSION ]]; then
    echo "VERSION environment not specified"
    exit 1
fi

mkdir -p $WORKINGDIR/release
cd $WORKINGDIR/release

for os in ${OS[@]}
do
    for arch in ${ARCH[@]}
    do
        OUTPUT="image-tools-$os-$arch-$VERSION"
        GOOS=$os GOARCH=$arch go build -ldflags "-s -w" -o $OUTPUT ..
        echo $OUTPUT
    done
done
