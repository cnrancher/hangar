#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/..
WORKINGDIR=$(pwd)

source $WORKINGDIR/scripts/version.sh

mkdir -p dist/artifacts
cp bin/hangar dist/artifacts/hangar${SUFFIX}
