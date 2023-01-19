#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

go test -v -cover --count=1 ./pkg/archive
go test -v -cover --count=1 ./pkg/archive/part
go test -v -cover --count=1 ./pkg/image
go test -v -cover --count=1 ./pkg/mirror
go test -v -cover --count=1 ./pkg/rancher/chart
go test -v -cover --count=1 ./pkg/rancher/listgenerator
go test -v -cover --count=1 ./pkg/registry
go test -v -cover --count=1 ./pkg/utils
