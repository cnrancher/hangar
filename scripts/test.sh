#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

# Ensure charts folder exists
function ensure_git_repo_cloned {
    REPO="${1:-""}"
    DIR="${2:-""}"
    if [[ -z "$REPO" || -z "$DIR" ]]; then
        return
    fi
    if [[ -d "$DIR" ]]; then
        # already exists
        return
    fi
    git clone --depth=1 $REPO $DIR
}
ensure_git_repo_cloned "https://github.com/cnrancher/pandaria-catalog.git" "pkg/rancher/chartimages/test/pandaria-catalog"
ensure_git_repo_cloned "https://github.com/cnrancher/system-charts.git" "pkg/rancher/chartimages/test/system-charts"
ensure_git_repo_cloned "https://github.com/rancher/charts.git" "pkg/rancher/chartimages/test/rancher-charts"

if [[ ! -e "pkg/rancher/kdmimages/test/data.json" ]]; then
    wget --no-check-certificate https://github.com/rancher/kontainer-driver-metadata/raw/dev-v2.7/data/data.json -O pkg/rancher/kdmimages/test/data.json
fi

go test -v -cover --count=1 ./pkg/archive
go test -v -cover --count=1 ./pkg/archive/part
go test -v -cover --count=1 ./pkg/image
go test -v -cover --count=1 ./pkg/mirror
go test -v -cover --count=1 ./pkg/rancher/chartimages
go test -v -cover --count=1 ./pkg/rancher/kdmimages
go test -v -cover --count=1 ./pkg/rancher/listgenerator
go test -v -cover --count=1 ./pkg/registry
go test -v -cover --count=1 ./pkg/utils
