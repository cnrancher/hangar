#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR=$(pwd)

if [[ -e ./scripts/$1.sh ]]; then
    ./scripts/"$1.sh"
else
    exec "$@"
fi

chown -R $DAPPER_UID:$DAPPER_GID .
