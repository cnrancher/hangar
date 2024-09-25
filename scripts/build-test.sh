#!/bin/bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

echo "-------------------------"
echo "Start build test."
echo "-------------------------"

if [[ ! -z ${DRONE_TAG:-} ]]; then
    echo "build test skipped on release tag"
    exit 0
fi

echo "Test build non-debug binary file."
DEBUG=0 ./scripts/build.sh > /dev/null
ls -alh bin/hangar*
bin/hangar version
echo "-------------------------"

echo "Test build debug binary file."
DEBUG=true ./scripts/build.sh > /dev/null
ls -alh bin/hangar*
bin/hangar version

echo "-------------------------"
echo "Build test PASSED."
echo "-------------------------"
rm bin/hangar
