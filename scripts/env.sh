#!/usr/bin/env bash
# This script has following environment variables:
# WORKINGDIR: project directory
# VERSION: git tag (or DRONE_TAG), HEAD-<COMMITHASH> if not found
# GITCOMMIT: git commit hash of this project, UNKNOW if not found
# BUILD_FLAG: version and commit flags when build this project

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

GITCOMMIT=$(git rev-parse HEAD || echo "")
if [[ -z "${GITCOMMIT}" ]]; then
    if [[ -n "${DRONE_COMMIT_SHA:-}" ]]; then
        echo "DRONE_COMMIT_SHA: ${DRONE_COMMIT_SHA}"
        GITCOMMIT=${DRONE_COMMIT_SHA}
    else
        GITCOMMIT="UNKNOW"
    fi
fi
echo "Git commit: ${GITCOMMIT}"

VERSION=$(git describe --tags 2>/dev/null || echo "")
if [[ -z "${VERSION}" ]]; then
    if [[ -n "${DRONE_TAG:-}" ]]; then
        echo "DRONE_TAG: ${DRONE_TAG}"
        VERSION=${DRONE_TAG}
    else
        VERSION="HEAD-${GITCOMMIT:0:8}"
    fi
fi
echo "Build version: ${VERSION}"

BUILD_FLAG=""
if ! echo ${DRONE_TAG:-} | grep -q "rc"; then
    BUILD_FLAG="-s -w"
fi

if [[ "${GITCOMMIT}" != "UNKNOW" ]]; then
    BUILD_FLAG="${BUILD_FLAG} -X 'github.com/cnrancher/hangar/pkg/utils.GitCommit=${GITCOMMIT}'"
fi
BUILD_FLAG="${BUILD_FLAG} -X 'github.com/cnrancher/hangar/pkg/utils.Version=${VERSION}'"
