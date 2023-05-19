#!/bin/bash

set -euo pipefail

# Need to launch a Harbor V2 server and specify
# its URL, Username, Password in following env variables.
export SOURCE_REGISTRY="${SOURCE_REGISTRY:-}"
export SOURCE_USERNAME="${SOURCE_USERNAME:-}"
export SOURCE_PASSWORD="${SOURCE_PASSWORD:-}"

export DEST_REGISTRY="${DEST_REGISTRY:-}"
export DEST_USERNAME="${DEST_USERNAME:-}"
export DEST_PASSWORD="${DEST_PASSWORD:-}"

export TEST_SKIP_TLS="${TEST_SKIP_TLS:-}"
