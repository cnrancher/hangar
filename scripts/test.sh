#!/usr/bin/env bash

set -e

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

go test -v -cover --count=1 ./pkg/utils
go test -v -cover --count=1 ./pkg/mirror
go test -v -cover --count=1 ./pkg/image
go test -v -cover --count=1 ./pkg/archive
go test -v -cover --count=1 ./pkg/archive/part