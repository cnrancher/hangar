#!/usr/bin/env bash

set -euo pipefail

if [[ $# -gt 0 ]]; then
    exec "$@"
else
    bash
fi
