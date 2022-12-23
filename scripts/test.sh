#!/usr/bin/env bash

set -e

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

go test -v -cover --count=1 ./utils
go test -v -cover --count=1 ./mirror
go test -v -cover --count=1 ./image