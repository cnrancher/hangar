#!/bin/bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

echo "-------------------------"
echo "Start build test."
echo "-------------------------"

echo "Test build non-debug binary file."
DEBUG=0 ./scripts/build.sh > /dev/null

echo "Test build debug binary file."
DEBUG=true ./scripts/build.sh > /dev/null

echo "Test build non-debug binary file with CGO disabled."
DISABLE_CGO=1 DEBUG=0 ./scripts/build.sh > /dev/null

echo "Test build debug binary file with CGO disabled."
DISABLE_CGO=1 DEBUG=true ./scripts/build.sh > /dev/null
echo "-------------------------"
echo "Build test PASSED."
echo "-------------------------"
rm bin/hangar