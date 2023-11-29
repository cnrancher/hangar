#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

# Ensure charts folder exists
function ensure_git_repo_cloned {
    REPO="${1:-""}"
    DIR="${2:-""}"
    BRANCH="${3-""}"
    if [[ -z "$REPO" || -z "$DIR" || -z "$BRANCH" ]]; then
        return
    fi
    if [[ -d "$DIR" ]]; then
        # already exists
        return
    fi
    git clone --branch $BRANCH --depth=1 $REPO $DIR
}
ensure_git_repo_cloned "https://github.com/cnrancher/pandaria-catalog.git" "pkg/rancher/chartimages/test/pandaria-catalog" "dev/v2.8"
ensure_git_repo_cloned "https://github.com/cnrancher/system-charts.git" "pkg/rancher/chartimages/test/system-charts" "dev-v2.8"
ensure_git_repo_cloned "https://github.com/rancher/charts.git" "pkg/rancher/chartimages/test/rancher-charts" "dev-v2.8"

if [[ ! -e "pkg/rancher/kdmimages/test/data.json" ]]; then
    wget --no-check-certificate https://github.com/rancher/kontainer-driver-metadata/raw/dev-v2.8/data/data.json -O pkg/rancher/kdmimages/test/data.json
fi

go test -cover --count=1 ./...
