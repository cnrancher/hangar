#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

${WORKINGDIR}/scripts/test.sh
${WORKINGDIR}/scripts/build-all.sh
