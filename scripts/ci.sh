#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

${WORKINGDIR}/scripts/validate.sh
${WORKINGDIR}/scripts/test.sh
${WORKINGDIR}/scripts/build.sh
${WORKINGDIR}/scripts/package.sh
