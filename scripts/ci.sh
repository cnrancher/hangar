#!/usr/bin/env bash

set -e

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

${WORKINGDIR}/scripts/test.sh
${WORKINGDIR}/scripts/build.sh
