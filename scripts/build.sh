#!/bin/bash

set -e

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

go version

go build -x -o image-tools .
