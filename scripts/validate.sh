#!/usr/bin/env bash

set -euo pipefail
cd $(dirname $0)/..
WORKINGDIR=$(pwd)

echo 'Running: go mod verify'
go mod verify
if [[ -n "$(git status --porcelain --untracked-files=no)" ]]; then
    echo 'go mod produced differences'
    exit 1
fi

echo 'Running: go fmt'
go fmt
if [[ -n "$(git status --porcelain --untracked-files=no)" ]]; then
    echo 'go fmt produced differences'
    exit 1
fi

echo 'Running: go generate'
go generate
if [[ -n "$(git status --porcelain --untracked-files=no)" ]]; then
    echo 'go generate produced differences'
    exit 1
fi

echo 'Running: go mod tidy'
go mod tidy
if [[ -n "$(git status --porcelain --untracked-files=no)" ]]; then
    echo 'go mod tidy produced differences'
    exit 1
fi

if type golangci-lint &> /dev/null; then
    echo Running: golangci-lint
    golangci-lint run
fi
