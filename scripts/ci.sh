#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

${WORKINGDIR}/scripts/verify.sh
${WORKINGDIR}/scripts/test.sh
${WORKINGDIR}/scripts/build-test.sh
${WORKINGDIR}/scripts/build.sh
${WORKINGDIR}/scripts/package.sh
